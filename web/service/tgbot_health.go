package service

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"x-ui/util/common"
)

// CE 路线图 #104：TG `/health` 命令——综合系统 + xray + 入站健康度 + Top5 流量报告。
//
// 设计原则：
//   - 严格本机：复用 serverService.GetStatus（gopsutil 本机指标）+ inboundService（DB）
//     + InboundHealthService（dial 127.0.0.1）；不发任何外部 API 请求
//   - 输出 markdown，TG 客户端原生渲染（粗体 / 列表 / 反引号）
//   - 失败容忍：任何子模块出错只影响该段，整体仍出 partial report 给运维
//
// timing：
//   - GetStatus 本身有 gopsutil 的 cpu.Percent(0) 立即采样（< 50ms）
//   - InboundHealthService.CheckAll(true) 强刷端口探测；
//     30 条入站并发 8 + 1s timeout ≈ 4 batch * 1s = 4s 上限（实际本机 dial 几乎 instant）
//   - 整体响应在 plan 验收要求的 3s 内（端口被占都满刷的极端情况例外）

// sendCEHealthReport 拉取并发送一份完整的健康度报告。
// 调用方：tgbot.go answerCommand 中 case "health"。
// 仅 admin 可调用（dispatcher 已校验 isAdmin）。
func (t *Tgbot) sendCEHealthReport(chatId int64) {
	status := t.serverService.GetStatus(t.lastStatus)

	inbounds, err := t.inboundService.GetAllInbounds()
	if err != nil {
		inbounds = nil
	}

	// 复用 #103 的 InboundHealthService。值字段未初始化也能跑：
	// 其方法均通过 inboundService 的全局 DB / xrayService 的全局 p 拿数据。
	healthSvc := InboundHealthService{}
	healths, _ := healthSvc.CheckAll(true)

	greenN, yellowN, redN, grayN := 0, 0, 0, 0
	var problems []HealthStatus
	for _, h := range healths {
		switch h.Status {
		case "green":
			greenN++
		case "yellow":
			yellowN++
			problems = append(problems, h)
		case "red":
			redN++
			problems = append(problems, h)
		case "gray":
			grayN++
		}
	}

	type trafficRow struct {
		remark string
		port   int
		bytes  int64
	}
	rows := make([]trafficRow, 0, len(inbounds))
	for _, ib := range inbounds {
		if !ib.Enable {
			continue
		}
		rows = append(rows, trafficRow{
			remark: ib.Remark,
			port:   ib.Port,
			bytes:  ib.Up + ib.Down,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].bytes > rows[j].bytes })
	if len(rows) > 5 {
		rows = rows[:5]
	}

	var sb strings.Builder
	sb.WriteString("🩺 *x-panel-ce 健康度报告*\n\n")

	sb.WriteString("⚙️ *系统*\n")
	sb.WriteString(fmt.Sprintf("  CPU: %.1f%% (%dc/%dt @ %.0fMHz)\n",
		status.Cpu, status.CpuCores, status.LogicalPro, status.CpuSpeedMhz))
	sb.WriteString(fmt.Sprintf("  MEM: %s / %s (%s)\n",
		common.FormatTraffic(int64(status.Mem.Current)),
		common.FormatTraffic(int64(status.Mem.Total)),
		percentString(status.Mem.Current, status.Mem.Total)))
	sb.WriteString(fmt.Sprintf("  Disk: %s / %s (%s)\n",
		common.FormatTraffic(int64(status.Disk.Current)),
		common.FormatTraffic(int64(status.Disk.Total)),
		percentString(status.Disk.Current, status.Disk.Total)))
	sb.WriteString(fmt.Sprintf("  Uptime: %s\n", formatUptimeSeconds(status.Uptime)))
	if len(status.Loads) >= 3 {
		sb.WriteString(fmt.Sprintf("  Load: %.2f / %.2f / %.2f\n", status.Loads[0], status.Loads[1], status.Loads[2]))
	}
	sb.WriteString(fmt.Sprintf("  Net: ↑%s/s  ↓%s/s\n",
		common.FormatTraffic(int64(status.NetIO.Up)),
		common.FormatTraffic(int64(status.NetIO.Down))))
	sb.WriteString(fmt.Sprintf("  TCP conns: %d / UDP: %d\n", status.TcpCount, status.UdpCount))

	sb.WriteString("\n🚀 *Xray*\n")
	sb.WriteString(fmt.Sprintf("  状态: `%s`\n", status.Xray.State))
	sb.WriteString(fmt.Sprintf("  版本: `%s`\n", status.Xray.Version))
	if status.Xray.ErrorMsg != "" {
		sb.WriteString(fmt.Sprintf("  ⚠️ 上次错误: `%s`\n", trimMultiline(status.Xray.ErrorMsg, 120)))
	}
	sb.WriteString(fmt.Sprintf("  面板进程内存: %s\n", common.FormatTraffic(int64(status.AppStats.Mem))))
	sb.WriteString(fmt.Sprintf("  面板进程 Uptime: %s\n", formatUptimeSeconds(status.AppStats.Uptime)))

	total := greenN + yellowN + redN + grayN
	sb.WriteString("\n📊 *入站健康度* (CE #103)\n")
	sb.WriteString(fmt.Sprintf("  🟢 %d / 🟡 %d / 🔴 %d / ⚪ %d  (total %d)\n",
		greenN, yellowN, redN, grayN, total))
	if len(problems) > 0 {
		sb.WriteString("  问题节点：\n")
		maxList := 10
		for i, p := range problems {
			if i >= maxList {
				sb.WriteString(fmt.Sprintf("   …（还有 %d 条已省略）\n", len(problems)-maxList))
				break
			}
			sb.WriteString(fmt.Sprintf("   • #%d port=%d → %s (%s)\n", p.Id, p.Port, p.Status, p.Reason))
		}
	}

	sb.WriteString("\n📈 *流量 Top 5*\n")
	if len(rows) == 0 {
		sb.WriteString("  (no traffic data)\n")
	} else {
		for i, r := range rows {
			label := r.remark
			if label == "" {
				label = fmt.Sprintf("port=%d", r.port)
			}
			sb.WriteString(fmt.Sprintf("  %d. %s → %s\n", i+1, label, common.FormatTraffic(r.bytes)))
		}
	}

	if status.PublicIP.IPv4 != "" && status.PublicIP.IPv4 != "N/A" {
		sb.WriteString("\n🌐 *公网*\n")
		sb.WriteString(fmt.Sprintf("  IPv4: `%s`\n", status.PublicIP.IPv4))
		if status.PublicIP.IPv6 != "" && status.PublicIP.IPv6 != "N/A" {
			sb.WriteString(fmt.Sprintf("  IPv6: `%s`\n", status.PublicIP.IPv6))
		}
	}

	sb.WriteString(fmt.Sprintf("\n_%s_", time.Now().Format("2006-01-02 15:04:05 MST")))

	t.SendMsgToTgbot(chatId, sb.String())
}

// percentString 把 used/total 渲染成 "30.0%" 文本，total=0 时返回 "n/a"
// 避免除零。
func percentString(used, total uint64) string {
	if total == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.1f%%", 100.0*float64(used)/float64(total))
}

// formatUptimeSeconds 秒数 → "12d 5h 33m" / "5h 12m" / "33m" 友好串。
// 不引入第三方包；仅 stdlib。
func formatUptimeSeconds(secs uint64) string {
	if secs == 0 {
		return "0m"
	}
	d := secs / 86400
	h := (secs % 86400) / 3600
	m := (secs % 3600) / 60
	switch {
	case d > 0:
		return fmt.Sprintf("%dd %dh %dm", d, h, m)
	case h > 0:
		return fmt.Sprintf("%dh %dm", h, m)
	default:
		return fmt.Sprintf("%dm", m)
	}
}

// trimMultiline 把多行字符串压成单行 + 截断至 maxLen，避免 TG 消息过长。
// 用于 Xray.ErrorMsg 之类可能含 stacktrace 的字段。
func trimMultiline(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " | ")
	if maxLen > 0 && len(s) > maxLen {
		// 注意：byte 截断对 UTF-8 中文不安全；用 rune 切。
		runes := []rune(s)
		if len(runes) > maxLen {
			s = string(runes[:maxLen]) + "…"
		}
	}
	return s
}
