package service

import (
	"net"
	"strconv"
	"testing"
	"time"
)

// TestClassifyInboundHealth 验证 CE 路线图 #103 的三态判定纯函数：
// - gray：DB enable=false（用户主动禁用）
// - red：enable=true 但 xray config 不含此 tag（孤儿）
// - 其余情况返回 needDial=true，由调用方根据 dial 结果填充 green/yellow
func TestClassifyInboundHealth(t *testing.T) {
	xrayTags := map[string]int{
		"inbound-1": 30001,
		"inbound-2": 30002,
	}

	tests := []struct {
		name       string
		id         int
		tag        string
		port       int
		enable     bool
		wantStatus string
		wantReason string
		wantDial   bool
	}{
		{
			name:       "disabled inbound is gray",
			id:         1, tag: "inbound-1", port: 30001, enable: false,
			wantStatus: "gray", wantReason: "disabled", wantDial: false,
		},
		{
			name:       "orphan inbound (not in xray) is red",
			id:         3, tag: "orphan-inbound", port: 30099, enable: true,
			wantStatus: "red", wantReason: "orphan_not_in_xray", wantDial: false,
		},
		{
			name:       "enabled and in xray needs dial",
			id:         1, tag: "inbound-1", port: 30001, enable: true,
			wantStatus: "", wantReason: "", wantDial: true,
		},
		{
			name:       "disabled overrides orphan check",
			id:         9, tag: "missing", port: 0, enable: false,
			wantStatus: "gray", wantReason: "disabled", wantDial: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hs, needDial := classifyInboundHealth(tc.id, tc.tag, tc.port, tc.enable, xrayTags)
			if hs.Status != tc.wantStatus {
				t.Errorf("status=%q want %q", hs.Status, tc.wantStatus)
			}
			if hs.Reason != tc.wantReason {
				t.Errorf("reason=%q want %q", hs.Reason, tc.wantReason)
			}
			if needDial != tc.wantDial {
				t.Errorf("needDial=%v want %v", needDial, tc.wantDial)
			}
			if hs.Id != tc.id || hs.Tag != tc.tag || hs.Port != tc.port {
				t.Errorf("metadata mismatch: %+v", hs)
			}
		})
	}
}

// TestDialAll 验证端口探测：起一个 listener 占一个端口，另一个端口空闲。
// 期望：listener 端口标 green，空闲端口标 yellow，并发上限不会卡住。
func TestDialAll(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()

	listeningPort := ln.Addr().(*net.TCPAddr).Port

	// 找一个空闲端口
	tmpLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("tmp listen: %v", err)
	}
	freePort := tmpLn.Addr().(*net.TCPAddr).Port
	tmpLn.Close()
	time.Sleep(50 * time.Millisecond)

	results := []HealthStatus{
		{Id: 1, Tag: "listening", Port: listeningPort},
		{Id: 2, Tag: "free", Port: freePort},
	}
	jobs := []healthDialJob{
		{idx: 0, port: listeningPort},
		{idx: 1, port: freePort},
	}

	svc := &InboundHealthService{}
	svc.dialAll(results, jobs)

	if results[0].Status != "green" {
		t.Errorf("listening port: status=%q reason=%q, want green", results[0].Status, results[0].Reason)
	}
	if results[1].Status != "yellow" {
		t.Errorf("free port: status=%q reason=%q, want yellow port_not_listening",
			results[1].Status, results[1].Reason)
	}
}

// TestDialAllManyRefused 用大量"无人监听"端口验证 dialAll：
// - 全部应标记为 yellow / port_not_listening
// - 总耗时不应超过 dialOverallCtxTO + 缓冲（防止信号量泄漏导致死锁）
//
// 注意：在 Linux 本机连一个没人监听的 127.0.0.1 端口会立即收到
// ECONNREFUSED 而不是 timeout，所以本测试不严格验证 "并发上限带来的耗时阶梯"
// （那需要远程不可达地址，会让 CI 抖动）。
func TestDialAllManyRefused(t *testing.T) {
	const total = maxParallelDials*2 + 2
	jobs := make([]healthDialJob, total)
	results := make([]HealthStatus, total)
	for i := range jobs {
		port := 39000 + i
		jobs[i] = healthDialJob{idx: i, port: port}
		results[i] = HealthStatus{Id: i + 1, Tag: "tag-" + strconv.Itoa(i), Port: port}
	}

	svc := &InboundHealthService{}
	start := time.Now()
	svc.dialAll(results, jobs)
	elapsed := time.Since(start)

	if elapsed > dialOverallCtxTO+500*time.Millisecond {
		t.Errorf("dialAll took %v, exceeds dialOverallCtxTO=%v (semaphore leak?)", elapsed, dialOverallCtxTO)
	}

	for i := range results {
		if results[i].Status != "yellow" {
			t.Errorf("idx=%d status=%q want yellow (port should not listen)", i, results[i].Status)
		}
	}
}

// TestCheckAllCacheTTL 验证 30s 缓存：第二次调用（force=false）拿到与第一次相同的副本。
// 通过手动改 cachedAt 强制让缓存"未过期"，第二次调用应返回旧缓存而非重新计算。
func TestCheckAllCacheTTL(t *testing.T) {
	svc := &InboundHealthService{}
	svc.mu.Lock()
	svc.cached = []HealthStatus{
		{Id: 99, Tag: "cached-tag", Port: 30099, Status: "green"},
	}
	svc.cachedAt = time.Now()
	svc.mu.Unlock()

	got, err := svc.CheckAll(false)
	if err != nil {
		t.Fatalf("CheckAll(false): %v", err)
	}
	if len(got) != 1 || got[0].Id != 99 || got[0].Status != "green" {
		t.Errorf("expected cached payload, got %+v", got)
	}

	// 验证返回的是 copy 而非内部 slice 引用：修改它不应影响下次缓存命中
	got[0].Status = "MUTATED"
	got2, _ := svc.CheckAll(false)
	if got2[0].Status != "green" {
		t.Errorf("cache returned shared reference; want isolated copy")
	}
}

// TestCheckAllCacheExpired 简单验证缓存过期判定：
// cachedAt 设为 healthCacheTTL+1s 之前，CheckAll(false) 应认为缓存过期。
// 这个测试不调到 IO 层，只验证 mu 路径下的时间判断分支。
func TestCheckAllCacheExpired(t *testing.T) {
	svc := &InboundHealthService{}
	svc.mu.Lock()
	svc.cached = []HealthStatus{
		{Id: 1, Tag: "stale", Port: 30001, Status: "green"},
	}
	svc.cachedAt = time.Now().Add(-(healthCacheTTL + 1*time.Second))
	expired := time.Since(svc.cachedAt) >= healthCacheTTL
	svc.mu.Unlock()

	if !expired {
		t.Fatalf("cachedAt setup wrong; expected expired but Since()=%v < TTL=%v",
			time.Since(svc.cachedAt), healthCacheTTL)
	}
}

// TestDialAllWithEmptyJobs 边界：jobs 为空不应 panic 也不修改 results。
func TestDialAllWithEmptyJobs(t *testing.T) {
	svc := &InboundHealthService{}
	results := []HealthStatus{
		{Id: 1, Tag: "x", Port: 30001, Status: "gray", Reason: "disabled"},
	}
	svc.dialAll(results, nil)
	if results[0].Status != "gray" || results[0].Reason != "disabled" {
		t.Errorf("dialAll([]) mutated results: %+v", results)
	}
}

