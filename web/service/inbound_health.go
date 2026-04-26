package service

import (
	"context"
	"net"
	"strconv"
	"sync"
	"time"
)

// CE 路线图 #103：入站健康度可视化（绿/黄/红/灰）。
//
// 三态判定（管理员视角，便于第一时间发现孤儿数据 / 端口异常）：
//   green  -> DB enable=true 且 xray 当前 config 含此 inbound 且端口可建连
//   yellow -> DB enable=true 且 xray 含此 inbound 但端口 dial 失败
//             （xray 启动失败 / 配置无效 / 端口被另一个崩溃中的进程占等）
//   red    -> DB enable=true 但 xray 当前 config 不含此 inbound
//             （孤儿；典型 ce-1.0.2 user_id=0 / 序列化失败 / xray 还没重启等）
//   gray   -> DB enable=false（用户主动禁用，不算异常）
//
// 节流策略（threading / timing）：
//   - 默认 healthCacheTTL=30s 内复用结果，前端轮询不会把后端打挂
//   - force=true 绕过缓存（管理员手动点刷新）
//   - 端口探测 sync.WaitGroup + 信号量 channel 限制并发上限 maxParallelDials=8，
//     避免几十条同时 dial 把 ulimit 打爆
//   - 每次 dial 1s timeout（dialTimeout）；总 ctx 5s 超时兜底
//   - 缓存读写用 sync.Mutex 保护，cached slice 读出前完整 copy 避免外部修改
//
// 实现要点：
//   - 端口探测纯 net.DialTimeout("tcp", "127.0.0.1:port")，不依赖 ss/lsof，
//     纯标准库、零外部命令、零网络外发
//   - 取 xray 当前运行 config 走 xrayService.GetCurrentXrayConfig()
//     （内存对象，零 IO；不调 GetXrayConfig 因为它会重建模板有副作用）
//   - 不调 inboundService.GetXrayConfig；不重启 xray
type InboundHealthService struct {
	inboundService InboundService
	xrayService    XrayService

	mu       sync.Mutex
	cached   []HealthStatus
	cachedAt time.Time
}

// HealthStatus 单条入站的健康度报告。
type HealthStatus struct {
	Id     int    `json:"id"`
	Tag    string `json:"tag"`
	Port   int    `json:"port"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

const (
	healthCacheTTL   = 30 * time.Second
	dialTimeout      = 1 * time.Second
	dialOverallCtxTO = 5 * time.Second
	maxParallelDials = 8
)

// CheckAll 返回当前所有 inbound 的健康度。
// force=false 时若缓存未过期直接返回缓存副本。
func (s *InboundHealthService) CheckAll(force bool) ([]HealthStatus, error) {
	s.mu.Lock()
	if !force && time.Since(s.cachedAt) < healthCacheTTL && s.cached != nil {
		out := make([]HealthStatus, len(s.cached))
		copy(out, s.cached)
		s.mu.Unlock()
		return out, nil
	}
	s.mu.Unlock()

	inbounds, err := s.inboundService.GetAllInbounds()
	if err != nil {
		return nil, err
	}

	xrayTags := s.collectXrayTags()

	results := make([]HealthStatus, len(inbounds))
	jobs := make([]healthDialJob, 0, len(inbounds))

	for i, inbound := range inbounds {
		hs, needDial := classifyInboundHealth(inbound.Id, inbound.Tag, inbound.Port, inbound.Enable, xrayTags)
		results[i] = hs
		if needDial {
			jobs = append(jobs, healthDialJob{idx: i, port: inbound.Port})
		}
	}

	if len(jobs) > 0 {
		s.dialAll(results, jobs)
	}

	s.mu.Lock()
	s.cached = make([]HealthStatus, len(results))
	copy(s.cached, results)
	s.cachedAt = time.Now()
	s.mu.Unlock()

	out := make([]HealthStatus, len(results))
	copy(out, results)
	return out, nil
}

// collectXrayTags 取 xray 进程当前 config 中所有 inbound 的 tag → port 映射。
// xray 未运行时返回空 map（所有 enable 入站会被判 red）。
func (s *InboundHealthService) collectXrayTags() map[string]int {
	out := make(map[string]int)
	if !s.xrayService.IsXrayRunning() {
		return out
	}
	cfg := s.xrayService.GetCurrentXrayConfig()
	if cfg == nil {
		return out
	}
	for _, ic := range cfg.InboundConfigs {
		if ic.Tag != "" {
			out[ic.Tag] = ic.Port
		}
	}
	return out
}

// classifyInboundHealth 把"三态判定"从 IO 中剥出来便于单测。
// 返回 (HealthStatus, needDial)：
//   - gray/red 立即终态，不需要 dial
//   - 其他情况（在 xrayTags 中）返回 (零状态, true)，由调用方填充 dial 结果
func classifyInboundHealth(id int, tag string, port int, enable bool, xrayTags map[string]int) (HealthStatus, bool) {
	hs := HealthStatus{Id: id, Tag: tag, Port: port}
	if !enable {
		hs.Status = "gray"
		hs.Reason = "disabled"
		return hs, false
	}
	if _, ok := xrayTags[tag]; !ok {
		hs.Status = "red"
		hs.Reason = "orphan_not_in_xray"
		return hs, false
	}
	return hs, true
}

// healthDialJob 端口探测任务（idx → 写回 results 的位置）。
type healthDialJob struct {
	idx  int
	port int
}

// dialAll 并发探测端口，并发上限 maxParallelDials。
// 单条失败不传播；按 idx 写回 results 切片对应位置。
func (s *InboundHealthService) dialAll(results []HealthStatus, jobs []healthDialJob) {
	sem := make(chan struct{}, maxParallelDials)
	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), dialOverallCtxTO)
	defer cancel()

	for _, j := range jobs {
		j := j
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(j.port))
			dialer := &net.Dialer{Timeout: dialTimeout}
			conn, err := dialer.DialContext(ctx, "tcp", addr)
			if err != nil {
				results[j.idx].Status = "yellow"
				results[j.idx].Reason = "port_not_listening"
				return
			}
			_ = conn.Close()
			results[j.idx].Status = "green"
		}()
	}
	wg.Wait()
}

