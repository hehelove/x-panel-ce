package service

import (
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// CE 路线图 #112：Cron 任务可视化。
//
// 痛点：流量重置 / TG 报告 / DB 备份这几个 cron 现在都是隐式的。
// 用户排查"为什么没自动重置"得读源码。本服务提供只读视图：
// 任务名 / cron 表达式 / 上次执行时间 / 下次执行时间，
// 让面板里能看到 robfig/cron 调度状态。
//
// 设计原则：
//   1. 加法不重构：现有 cron.AddJob/AddFunc 调用点不动，
//      在 web.go 里只是改成调 Server.trackedAdd(name, spec, job)
//      / trackedAddFunc(name, spec, fn)，内部包一层后再 Track。
//   2. 只读：前端不允许修改 cron（避免误操作打挂面板），
//      也不暴露删除/暂停接口。
//   3. 零外部依赖：纯标准库 + robfig/cron（已在 go.mod）。
//
// threading：
//   - 元数据 map 由 sync.RWMutex 保护
//   - List() 从 cron.Cron.Entries() 取快照（cron 包内部已加锁）

// CronJobInfo 单条 cron 任务的可视化信息。
type CronJobInfo struct {
	ID       int       `json:"id"`
	Name     string    `json:"name"`
	Spec     string    `json:"spec"`
	Prev     time.Time `json:"prev"`     // 上次执行时间；零值表示"从未执行"
	Next     time.Time `json:"next"`     // 下次预计执行时间
	PrevText string    `json:"prevText"` // "5m ago" / "never"
	NextText string    `json:"nextText"` // "in 23h"
}

// CronInspector 全局单例：把 cron.EntryID -> 元数据（name, spec）映射起来。
// 由 web.go startTask 通过 trackedAdd 写入；只读 API 由 controller 调 List()。
type CronInspector struct {
	c    *cron.Cron
	mu   sync.RWMutex
	meta map[cron.EntryID]cronJobMeta
}

type cronJobMeta struct {
	Name string
	Spec string
}

// GlobalCronInspector 全局单例。web.go Start 阶段调 Bind 注入 *cron.Cron。
var GlobalCronInspector = &CronInspector{
	meta: make(map[cron.EntryID]cronJobMeta),
}

// Bind 注入 *cron.Cron。重复调用以最后一次为准（重启场景）。
// 调用方：web.go Server.Start。
func (ci *CronInspector) Bind(c *cron.Cron) {
	ci.mu.Lock()
	ci.c = c
	// 重新 bind 时清空元数据，避免旧 EntryID 残留误报
	ci.meta = make(map[cron.EntryID]cronJobMeta)
	ci.mu.Unlock()
}

// Track 关联一个 cron.EntryID 到 (name, spec) 元数据。
// id == 0 通常表示 AddJob/AddFunc 失败，跳过避免误记。
func (ci *CronInspector) Track(id cron.EntryID, name, spec string) {
	if id == 0 {
		return
	}
	ci.mu.Lock()
	if ci.meta == nil {
		ci.meta = make(map[cron.EntryID]cronJobMeta)
	}
	ci.meta[id] = cronJobMeta{Name: name, Spec: spec}
	ci.mu.Unlock()
}

// List 返回当前所有 cron 任务的快照，按 Next 升序（最快要跑的在前）。
// 没有 *cron.Cron 时（极端早期初始化阶段）返回空切片。
func (ci *CronInspector) List() []CronJobInfo {
	ci.mu.RLock()
	c := ci.c
	metaSnap := make(map[cron.EntryID]cronJobMeta, len(ci.meta))
	for k, v := range ci.meta {
		metaSnap[k] = v
	}
	ci.mu.RUnlock()

	if c == nil {
		return []CronJobInfo{}
	}

	now := time.Now()
	entries := c.Entries()
	out := make([]CronJobInfo, 0, len(entries))
	for _, e := range entries {
		m := metaSnap[e.ID]
		name := m.Name
		spec := m.Spec
		if name == "" {
			name = "(unnamed)"
		}
		if spec == "" {
			spec = "(spec unknown)"
		}
		info := CronJobInfo{
			ID:   int(e.ID),
			Name: name,
			Spec: spec,
			Prev: e.Prev,
			Next: e.Next,
		}
		info.PrevText = relativeTimeText(e.Prev, now, true /*past*/)
		info.NextText = relativeTimeText(e.Next, now, false /*future*/)
		out = append(out, info)
	}

	sort.SliceStable(out, func(i, j int) bool {
		// Zero Next time 排到最后
		if out[i].Next.IsZero() && !out[j].Next.IsZero() {
			return false
		}
		if !out[i].Next.IsZero() && out[j].Next.IsZero() {
			return true
		}
		return out[i].Next.Before(out[j].Next)
	})
	return out
}

// relativeTimeText 把 t 渲染为 "5m ago" / "in 23h" / "never" 等友好串。
// past=true 时零值返回 "never"；past=false 时零值返回 "n/a"。
func relativeTimeText(t time.Time, now time.Time, past bool) string {
	if t.IsZero() {
		if past {
			return "never"
		}
		return "n/a"
	}
	d := t.Sub(now)
	abs := d
	if abs < 0 {
		abs = -abs
	}
	var unit string
	switch {
	case abs < time.Minute:
		s := int(abs.Seconds())
		unit = formatPlural(s, "s")
	case abs < time.Hour:
		s := int(abs.Minutes())
		unit = formatPlural(s, "m")
	case abs < 24*time.Hour:
		s := int(abs.Hours())
		unit = formatPlural(s, "h")
	default:
		s := int(abs / (24 * time.Hour))
		unit = formatPlural(s, "d")
	}
	if d < 0 {
		return unit + " ago"
	}
	return "in " + unit
}

func formatPlural(n int, suffix string) string {
	return strconv.Itoa(n) + suffix
}
