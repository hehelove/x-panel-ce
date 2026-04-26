package job

import (
	"x-ui/logger"
	"x-ui/web/service"
)

// CE 路线图 #15：流量自动周期重置 job。
//
// 由 web.go 注册为 @daily（每日 0:00）触发，遍历所有客户端，根据其
// resetCycle 字段（在 client JSON 内）判断是否到期：
//   0  = 从不
//   1  = 每日
//   7  = 仅周一
//   30 = 仅每月 1 号
// 重置只清流量计数（up/down 归 0、enable 复位），不动 xray 配置，
// 不重启 xray，对在线连接零打扰。
type ResetTrafficCycleJob struct {
	inboundService service.InboundService
}

func NewResetTrafficCycleJob() *ResetTrafficCycleJob {
	return new(ResetTrafficCycleJob)
}

func (j *ResetTrafficCycleJob) Run() {
	count, err := j.inboundService.ResetClientTrafficByCycle()
	if err != nil {
		logger.Warning("CE #15: 流量周期重置 job 出错:", err)
		return
	}
	if count > 0 {
		logger.Infof("CE #15: 流量周期重置 job 完成，本次共重置 %d 个客户端", count)
	}
}
