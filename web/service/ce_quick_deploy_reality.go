package service

// CE 路线图 #31 + #13：批量部署 N 条 VLESS+TCP+Reality+Vision 入站。
//
// 设计要点：
//   - 端口冲突检测：先扫描 portStart 之后的 N 个空闲端口，扫描上限受
//     CEQuickDeployRealityMaxScan 限制，避免恶意请求长时间扫描
//   - 每条 inbound 独立 X25519 keypair（调用 ServerService 的 xray 子
//     命令），保证一条泄漏不影响其余
//   - 补偿删除：任一条目失败立刻把已落库的全部回滚（DelInbound），
//     而不开外层事务以避免与 AddInbound 内部事务嵌套
//   - 默认参数（target/SNI）选用 #3 候选池中最稳定的 tesla.com，与
//     上游 RealityStreamSettings 默认一致；用户在 modal 内可覆盖
//   - 不联网：所有候选清单与默认值均硬编码，不向 GitHub/CDN 拉取

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"x-ui/database/model"
	"x-ui/logger"
	"x-ui/util/common"
)

const (
	CEQuickDeployRealityMaxCount         = 50
	CEQuickDeployRealityMaxScan          = 5000
	CEQuickDeployRealityDefaultPortStart = 30000
	CEQuickDeployRealityDefaultRemark    = "ce-reality"
	CEQuickDeployRealityDefaultTarget    = "www.tesla.com:443"
	CEQuickDeployRealityDefaultSNI       = "www.tesla.com"
)

type CEQuickDeployRealityRequest struct {
	Count        int    `json:"count" form:"count"`
	PortStart    int    `json:"portStart" form:"portStart"`
	RemarkPrefix string `json:"remarkPrefix" form:"remarkPrefix"`
	Target       string `json:"target" form:"target"`
	ServerNames  string `json:"serverNames" form:"serverNames"`
}

type CEQuickDeployRealityResponse struct {
	SuccessCount int   `json:"successCount"`
	InboundIDs   []int `json:"inboundIds"`
	Ports        []int `json:"ports"`
}

// QuickDeployReality 批量创建 VLESS+TCP+Reality+Vision 入站。
// 失败时自动补偿删除已成功的条目，对调用方保证"全成或全无"语义。
func (s *InboundService) QuickDeployReality(req CEQuickDeployRealityRequest) (*CEQuickDeployRealityResponse, error) {
	if err := normalizeQuickDeployRealityReq(&req); err != nil {
		return nil, err
	}

	ports, err := s.findFreePorts(req.PortStart, req.Count)
	if err != nil {
		return nil, err
	}

	deployedIDs := make([]int, 0, req.Count)
	rollback := func() {
		for _, id := range deployedIDs {
			if _, derr := s.DelInbound(id); derr != nil {
				logger.Warningf("CE #31: 补偿删除 inbound %d 失败: %v", id, derr)
			}
		}
	}

	for i := 0; i < req.Count; i++ {
		port := ports[i]
		remark := fmt.Sprintf("%s-%d", req.RemarkPrefix, port)
		inbound, ierr := buildRealityInbound(port, remark, req.Target, req.ServerNames)
		if ierr != nil {
			rollback()
			return nil, fmt.Errorf("build inbound %d/%d (port=%d): %w", i+1, req.Count, port, ierr)
		}
		saved, _, aerr := s.AddInbound(inbound)
		if aerr != nil {
			rollback()
			return nil, fmt.Errorf("add inbound %d/%d (port=%d): %w", i+1, req.Count, port, aerr)
		}
		deployedIDs = append(deployedIDs, saved.Id)
		logger.Infof("CE #31: 已部署 inbound %d (port=%d, remark=%s)", saved.Id, port, remark)
	}

	return &CEQuickDeployRealityResponse{
		SuccessCount: len(deployedIDs),
		InboundIDs:   deployedIDs,
		Ports:        ports,
	}, nil
}

// normalizeQuickDeployRealityReq 校验并填充默认值，所有越界值抬升为可
// 接受范围内的值，避免前端传 0/负数 触发隐藏 bug。
func normalizeQuickDeployRealityReq(req *CEQuickDeployRealityRequest) error {
	if req.Count <= 0 {
		req.Count = 10
	}
	if req.Count > CEQuickDeployRealityMaxCount {
		return common.NewErrorf("count 超过上限 %d", CEQuickDeployRealityMaxCount)
	}
	if req.PortStart < 1024 {
		req.PortStart = CEQuickDeployRealityDefaultPortStart
	}
	if req.PortStart > 65535-req.Count {
		return common.NewErrorf("portStart 过大，剩余端口空间不足")
	}
	req.RemarkPrefix = strings.TrimSpace(req.RemarkPrefix)
	if req.RemarkPrefix == "" {
		req.RemarkPrefix = CEQuickDeployRealityDefaultRemark
	}
	req.Target = strings.TrimSpace(req.Target)
	if req.Target == "" {
		req.Target = CEQuickDeployRealityDefaultTarget
	}
	req.ServerNames = strings.TrimSpace(req.ServerNames)
	if req.ServerNames == "" {
		req.ServerNames = CEQuickDeployRealityDefaultSNI
	}
	return nil
}

// findFreePorts 从 start 开始线性扫描，返回 count 个连续未被占用的端口。
// 扫描上限受 CEQuickDeployRealityMaxScan 限制，超出即报错；listen 固定
// 视为通配（""），与 AddInbound 默认行为一致。
func (s *InboundService) findFreePorts(start, count int) ([]int, error) {
	ports := make([]int, 0, count)
	maxEnd := start + CEQuickDeployRealityMaxScan
	if maxEnd > 65535 {
		maxEnd = 65535
	}
	for p := start; p <= maxEnd && len(ports) < count; p++ {
		used, err := s.checkPortExist("", p, 0)
		if err != nil {
			return nil, err
		}
		if !used {
			ports = append(ports, p)
		}
	}
	if len(ports) < count {
		return nil, common.NewErrorf("从 %d 起最多扫描 %d 个端口仍找不到 %d 个空闲槽位", start, CEQuickDeployRealityMaxScan, count)
	}
	return ports, nil
}

// randomHex 返回长度为 n（字节数）的安全随机十六进制字符串。
func randomHex(n int) (string, error) {
	if n <= 0 {
		return "", nil
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// randomLowerAndNum 返回长度为 n 的小写字母 + 数字组合（用于 email/subId）。
func randomLowerAndNum(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := 0; i < n; i++ {
		b[i] = charset[common.RandomInt(len(charset))]
	}
	return string(b)
}

// realityShortIds 返回 8 个长度递增的 hex shortId（与上游前端一致）。
func realityShortIds() ([]string, error) {
	lengths := []int{2, 4, 6, 8, 10, 12, 14, 16}
	out := make([]string, len(lengths))
	for i, n := range lengths {
		s, err := randomHex(n / 2)
		if err != nil {
			return nil, err
		}
		out[i] = s
	}
	return out, nil
}

// buildRealityInbound 拼装一条 VLESS+TCP+Reality+Vision 入站的 model.Inbound，
// settings/streamSettings/sniffing 三个字段直接构造为 JSON 字符串。
// 所有 client 字段保持与 #15/#30 兼容（resetCycle=0, resetDay=1）。
func buildRealityInbound(port int, remark, target, sni string) (*model.Inbound, error) {
	pair, err := (&ServerService{}).GetNewX25519Cert()
	if err != nil {
		return nil, fmt.Errorf("生成 X25519 keypair 失败: %w", err)
	}
	pairMap, ok := pair.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("X25519 keypair 类型异常")
	}
	privKey, _ := pairMap["privateKey"].(string)
	pubKey, _ := pairMap["publicKey"].(string)
	if privKey == "" || pubKey == "" {
		return nil, fmt.Errorf("X25519 keypair 字段为空")
	}

	shortIds, err := realityShortIds()
	if err != nil {
		return nil, err
	}

	clientID := uuid.NewString()
	email := randomLowerAndNum(8)
	subID := randomLowerAndNum(16)

	settings := map[string]any{
		"clients": []map[string]any{
			{
				"id":         clientID,
				"flow":       "xtls-rprx-vision",
				"email":      email,
				"limitIp":    0,
				"totalGB":    0,
				"expiryTime": 0,
				"enable":     true,
				"tgId":       "",
				"subId":      subID,
				"comment":    "",
				"reset":      0,
				"resetCycle": 0,
				"resetDay":   1,
			},
		},
		"decryption": "none",
		"fallbacks":  []any{},
	}

	streamSettings := map[string]any{
		"network":       "tcp",
		"security":      "reality",
		"externalProxy": []any{},
		"realitySettings": map[string]any{
			"show":         false,
			"xver":         0,
			"target":       target,
			"serverNames":  []string{sni},
			"privateKey":   privKey,
			"minClientVer": "",
			"maxClientVer": "",
			"maxTimediff":  0,
			"shortIds":     shortIds,
			"settings": map[string]any{
				"publicKey":   pubKey,
				"fingerprint": "chrome",
				"serverName":  "",
				"spiderX":     "/",
			},
		},
		"tcpSettings": map[string]any{
			"acceptProxyProtocol": false,
			"header": map[string]any{
				"type": "none",
			},
		},
	}

	sniffing := map[string]any{
		"enabled":      false,
		"destOverride": []string{"http", "tls", "quic", "fakedns"},
		"metadataOnly": false,
		"routeOnly":    false,
	}

	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}
	streamSettingsJSON, err := json.Marshal(streamSettings)
	if err != nil {
		return nil, err
	}
	sniffingJSON, err := json.Marshal(sniffing)
	if err != nil {
		return nil, err
	}

	return &model.Inbound{
		Up:             0,
		Down:           0,
		Total:          0,
		Remark:         remark,
		Enable:         true,
		ExpiryTime:     0,
		Listen:         "",
		Port:           port,
		Protocol:       model.VLESS,
		Settings:       string(settingsJSON),
		StreamSettings: string(streamSettingsJSON),
		Tag:            fmt.Sprintf("inbound-%d", port),
		Sniffing:       string(sniffingJSON),
	}, nil
}
