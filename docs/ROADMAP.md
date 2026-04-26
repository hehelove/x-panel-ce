# x-panel-ce CE 路线图（ROADMAP）

本文件是 [README](../README.md) 中"CE 路线图"段落（编号 1-32）的**结构化版本**：
对每一项标注 *上游源码坐标 / 依赖评估 / CE 策略 / 所属 Stage / 备注*，
用作后续 Stage 2-6 实施时的**唯一锚点**。

> 与 [`NOTICE.md`](../NOTICE.md) 第 4 节"与上游的差异"保持一致：
> 本仓库不实现 #26 / #27 / #28（积分体系）和 #32（购买机器人）。

---

## 1. 路线图与现实状态

- **来源**：上游 [`xeefei/X-Panel`](https://github.com/xeefei/X-Panel) 在 `README` / 安装脚本 / 面板 UI 中宣称的 31 条新增/优化项 + 1 条不实现项。
- **CE 取舍**：
  - **不实现**：#26、#27、#28（积分），#32（购买机器人） —— 与"无收款 / 无授权码"原则冲突。
  - **重定义**：#17（授权验证）、#24（首页会员等级）、#4（授权报告） —— 上游与"授权码销售"绑定，CE 中将剥离收款语义，仅保留**纯技术性**指纹/版本/部署校验，或直接删除（待 Stage 6 决定）。
  - **直接开源重写或增强**：其余 22 项。

> Stage 0 在本仓库已完成（NOTICE 添加、README/安装脚本/CI 去商业化），
> 详见 git log `a336dcd0` / `26b9504b` / `d0c983d2` 三个本地提交。

---

## 2. Stage 总览

| Stage | 主题 | 路线图编号 | 状态 | 依赖前置 |
|---|---|---|---|---|
| **0** | 仓库合规化 + 隐私后门紧急清理 | — | ✓ 已完成（commits a336dcd0 / 26b9504b / d0c983d2 / 9c5599d2） | — |
| **0.5** | 抽奖框架整段清理（隐私后门延续治理） | — | ✓ 已完成（commit 8c8d702e） | Stage 0 |
| **1** | 路线图文档化（本文件） | — | ✓ 已完成 | Stage 0 |
| **2** | 安装脚本 CE 化 | 5 / 7 / 8 / 9 / 21 / 22 / 29 | ✓ 已完成（详见 §3.1） | Stage 1 |
| **3** | TG Bot 通知与状态显示 | 2 / 4* / 6 / 10 / 11 / 12 / 16 / 19 / 25 | ✓ 已完成（详见 §3.2） | Stage 2 |
| **4** | 面板后台 UI / 入站增强 | 1 / 3 / 13 / 14 / 15 / 24* / 30 / 31 | ✓ 已完成（详见 §3.3） | Stage 3 |
| **5** | 主从与中转高级功能 | 18 / 20 / 23 | ✓ 决策化收尾（长期项目，commit 18fb1614） | Stage 4 |
| **6** | 授权验证机制改造 | 17* | ✓ 决策化收尾（完全移除 + 永久反遥测，commit 18fb1614） | 全部前置完成后单独评审 |
| **7** | 易用性 / 可观测性 / 运维增强 | 100-114（CE 自有，共 15 项） | 🚧 进行中（ce-1.1.0：#100 / #101 / #102 已发布；ce-1.2.0：#103 / #104 / #112 已发布；详见 [`docs/plans/2026-04-26-stage7-usability.md`](plans/2026-04-26-stage7-usability.md)） | Stage 0-6 完成后启动，按 ce-1.1.0 / 1.2.0 / 1.3.0 / 1.4.0 切分发布 |

> 标注 `*` 的条目在 CE 中需要**重定义而不是直接照搬**，详见下方明细。
> 首个完整 release 标签：[`ce-1.0.0`](https://github.com/hehelove/x-panel-ce/releases/tag/ce-1.0.0)

---

## 3. 逐项明细（保留 README 原编号 1-32）

> 表格读法：
> - **上游源码坐标**：本 fork 当前已有的、最可能承载该功能的入口文件/目录。Stage 实施前会在该位置做精确 grep 定位。
> - **CE 策略**：`OSS 重写` / `OSS 增强` / `中性化` / `不实现` / `待决策`。
> - **依赖**：是否依赖上游闭源 Pro 资源（如远程授权服务器、私有 sublink 仓库等）。

### 3.1 Stage 2 — 安装脚本 CE 化

> **菜单宿主已确认**：路线图原文所说的"第 26/27/28/29 选项"都在
> `x-ui.sh:show_menu`（L2060-2192，当前菜单为 0-25）。
> Stage 2 的菜单扩展工作集中在 `x-ui.sh`，`install.sh` 仅做头部注释收尾。

| # | 标题摘要 | 上游源码坐标 | 依赖评估 | CE 策略 | 备注 |
|---|---|---|---|---|---|
| 5 | 安装脚本界面增加 CE 明确标识 | `install.sh` 头注释 + `x-ui.sh:show_menu` banner | 无 | OSS 增强 | `install.sh` 第 4-5 行残余注释 `(付费/免费二合一)` 在 Stage 2.1 已清理；`x-ui.sh:show_menu` 头部 banner 已在 Stage 0 改为"x-panel-ce"。 |
| 7 | 第 26 选项部署"网页版 SSH 工具" | `x-ui.sh:show_menu` 新增 case 26 + 新函数 `webssh_install` | 第三方：[huashengdun/webssh](https://github.com/huashengdun/webssh)（**已选定**，MIT，Python tornado） | OSS 重写 | 部署方式：apt 装 python3-pip → `pip install webssh` → systemd unit 化（监听 127.0.0.1，由 nginx/反代或 ssh tunnel 暴露）；不开放公网默认端口。 |
| 8 | 第 27 选项"线路和 IP 质量检测" | `x-ui.sh:show_menu` 新增 case 27 + 新脚本 `linejc.sh`（顶层，仿 `dnsjc.sh`） | 公共数据源 | OSS 重写 | 数据源：Cloudflare trace、ipinfo.io 公开端点、本地 mtr/traceroute；**不接入任何商业测速 API**。 |
| 9 | 第 28 选项"地区服务器 DNS 检测" | `dnsjc.sh`（已存在）+ `x-ui.sh:show_menu` 新增 case 28 接入 | 无 | OSS 增强 | Stage 0 残留 banner "X-Panel-Pro 面板" 在 Stage 2.1 已修复。 |
| 21 | 证书申请"备用方式" | `x-ui.sh:ssl_cert_issue_main` 选项 5 → `ssl_cert_issue_standalone_embedded` (L824) | acme.sh | **已实现** | 上游已经实现 80 端口 standalone 申请，含 CA 选择（Let's Encrypt / Buypass / ZeroSSL）+ 防火墙临时放行；Stage 2.2.E 已核对代码完整性，未做行为变更。 |
| 22 | 自定义证书路径 | `x-ui.sh:ssl_cert_issue_main` 选项 4 (L1076-1144) | 无 | **已实现** | 上游已经实现：用户输入域名 + fullchain + privkey 路径 → 复制到 `/root/cert/<domain>/` → 调用 `x-ui cert` 应用 → restart；含空文件检查；Stage 2.2.E 已核对代码完整性。 |
| 29 | 第 29 选项"深度调优脚本"（BBR+FQ / TFO / 缓冲区 / 队列） | `x-ui.sh:show_menu` 新增 case 29 + 新函数 `tuning_kernel` | 无 | OSS 重写 | **应用策略已锁定**：dry-run 预览 → 备份 `/etc/sysctl.conf` 到 `/etc/sysctl.conf.bak.<ts>` → 用户确认（默认 N）→ 写入 `/etc/sysctl.d/99-x-panel-ce-tuning.conf`，提供回滚菜单项。 |

### 3.2 Stage 3 — TG Bot 通知与状态

| # | 标题摘要 | 上游源码坐标 | 依赖评估 | CE 策略 | 备注 |
|---|---|---|---|---|---|
| 2 | TG 端"版本更新"提示增加详细更新说明 | `web/service/tgbot.go` 中版本检查相关命令 | GitHub Releases API | OSS 增强 | 改读 `hehelove/x-panel-ce` 的 release notes |
| 4* | TG"发送授权报告"+ 唯一防伪码 | `web/service/tgbot.go` 中 `SendReport`/授权类回调 | 上游：远程授权服务器（已切断） | **重定义** | CE 改为"部署自检报告"：发送当前版本 / 节点入站统计 / 系统指标，**移除授权码字段** |
| 6 | TG 显示方式 + 会员标识 | `web/service/tgbot.go` 消息模板 | 无 | **重定义** | "会员标识"改为"角色标签"（admin / user），不再绑定会员等级；仅保留显示风格优化 |
| 10 | TG 端同步"网页版 SSH 工具"安装 | `web/service/tgbot.go` 命令路由 | Stage 2 #7 必须先完成 | OSS 重写 | 远程触发 Stage 2 提供的 webssh 安装脚本 |
| 11 | TG"服务器状态"版本号显示优化 | `web/service/tgbot.go` 状态命令 | 无 | OSS 增强 | 显示 `x-panel-ce vX.Y.Z`，去掉上游 Pro 特有标签 |
| 12 | TG 命令 `/webssh` | `web/service/tgbot.go` 命令注册 | Stage 2 #7 | OSS 重写 | 与 #10 联动 |
| 16 | TG"每日报告"可定制内容 + 周期 | `web/service/tgbot.go` 中定时任务 | 无 | OSS 增强 | 用 `gorm` 增加 `tg_report_pref` 表存用户偏好；定时任务用现有 `robfig/cron/v3` |
| 19 | TG"获取节点链接"（本机 + 远程被控端） | `web/service/tgbot.go` 节点查询命令 | Stage 5 主从机制（远端） | OSS 重写 | 第一阶段只做本机；远程版放 Stage 5 联动 |
| 25 | 节点上下线 TG 通知 | `web/service/tgbot.go` + xray 进程监控 | 无 | OSS 重写 | 复用 xray 现有进程信号；增加 inbound enable/disable 钩子 |

### 3.3 Stage 4 — 面板后台 UI / 入站增强

| # | 标题摘要 | 上游源码坐标 | 依赖评估 | CE 策略 | 备注 |
|---|---|---|---|---|---|
| 1 | 面板后台 UI 添加"X-Panel-Pro"标识 | `web/html/index.html` / `navigation.html` | 无 | **重定义** | CE 改为显示 `x-panel-ce` + GPL-3.0 角标，不使用 "Pro" 字样 |
| 3 | Reality 协议偷的域名"随机更换" | `web/html/modals/`（入站编辑模态）+ `xray` 配置生成 | 无 | OSS 重写 | 前端按钮 + 后端候选池（首批用公开常用 SNI 列表） |
| 13 | "一键配置"友好提示 | `web/html/modals/` + `web/service/tgbot.go` 中 onekey 回调 | 上游：曾跳转购买机器人（已中性化） | OSS 重写 | 改为生成本机 VLESS+Reality 入站，纯本地逻辑 |
| 14 | 首页 UI 5 主题（标准/炫彩/深海/暮光/幽林） | `web/html/index.html` + `web/assets/css/` | 无 | OSS 重写 | 引入 `localStorage` + 用户偏好持久化；首批做 2-3 个主题，其余迭代补 |
| 15 | 入站"重置流量"方式可视化（每日/每周/按月/从不） | `web/html/inbounds.html` + `database/model/inbound.go` | 无 | OSS 增强 | 字段层在上游已有 `expiryTime`/`reset`，前端增加可选 UI |
| 24* | 首页"会员等级"显示 | `web/html/index.html` | 上游：授权码绑定 | **重定义** | CE 改为显示"部署 ID + GPL-3.0 + 上游致谢"，不展示等级 |
| 30 | "每月重置流量"按指定 1-31 号 | `database/model/` + 重置定时任务 | 无 | OSS 重写 | 注意时区与月底 31 号对齐策略（28/29/30 月份要 fallback） |
| 31 | 批量部署 10 条 VLESS+TCP+Reality+Vision | `web/html/modals/` + `web/service/inbound.go` | 无 | OSS 重写 | 端口冲突检测 + 事务式批量插入 |

### 3.4 Stage 5 — 主从与中转高级（**决策：长期项目，不在 Stage 4 收尾后立刻动代码**）

> 状态：Stage 4 完成后评估；这三项每项都需要新增数据库表 / SSH 远程执行 / 跨机事务，
> 不适合在单会话内"一次推完"。当前 `web/html/servers.html` 已展示"功能开发中"
> 中性提示，前端入口保留以便未来贡献者直接接 OSS 实现，不向上游 Pro 引导购买。

| # | 标题摘要 | CE 当前状态 | 立项前提 | 拆分建议 |
|---|---|---|---|---|
| 18 | TG"多面板管理"（一个 bot 控多 VPS） | servers.html 表单已存在，submitServer 桩函数显示"功能开发中"；后端无 Server 表 | 决定 RPC 协议：HTTP+API token / SSH / WebSocket；凭据加密落库方案 | 第 1 步建 `Server` 表 + 列表 CRUD（不远程通信）；第 2 步加远程拉取被控端流量；第 3 步加 TG 命令路由 |
| 20 | "一键部署中转节点"（远端 Socks5 → 本机路由 → 二维码） | servers.html 中转区已有 UI，setupRelay 桩函数停在前端 | 依赖 #18 完成 Server 表；需要 SSH 库 + 命令注入防护 + 超时 | 第 1 步只生成本机入口（无远端 SSH）；第 2 步加 ssh 远端配置下发；第 3 步加生成二维码 |
| 23 | 数据快照 + 远程急救还原 | 暂无 UI/CLI 入口 | 决定快照内容范围（DB / sysctl / xray 配置）；签名/校验策略 | 第 1 步本机 tar.gz 备份 + sha256；第 2 步加 CLI 子命令；第 3 步对接 #18 Server 做远程下发 |

> **如果 CE 用户暂时需要多面板能力**：建议同时部署多套独立 x-panel-ce
> + 各自独立 TG bot；通过 TG 群组手动汇总。该方案安全边界清晰、不
> 引入跨机故障域。

### 3.5 Stage 6 — 授权验证机制（**决策：完全移除路径，不实现遥测**）

| # | 标题摘要 | CE 当前状态 | CE 决策 |
|---|---|---|---|
| 17* | 授权码"后台联网验证"+ 机器指纹 | Stage 0.1 已删除上游硬编码 Telegram 中央上报后门（commit `9c5599d2`）；HWID/授权码相关代码已无运行入口 | **完全移除**：CE 永不引入"匿名遥测/部署 ID 上报"，与 GPL-3.0 + 用户自托管定位一致 |

> **决策依据**：
> 1. CE 不收费 → 授权机制本质失效
> 2. Stage 0.1 揭露的上游"硬编码 Telegram bot token + chat_id 中央上报"
>    属于隐私后门，CE 必须以"零远程上报"为底线
> 3. 即便是"匿名 + opt-out 遥测"也会破坏用户信任（参考 Audacity 收购
>    后的争议）；任何遥测路径在 CE 中均**不会被开启**
>
> 如未来确实需要"管理员主动汇报安装统计"，请走 GitHub Issue / TG
> 群手动反馈，**不允许**面板代码自动联网。

### 3.6 明确不实现

| # | 标题摘要 | CE 策略 | 备注 |
|---|---|---|---|
| 26 | 面板"签到得积分" | **不实现** | 用户明确要求跳过 |
| 27 | TG 签到积分 / 查询 / 换购 / 排行榜 | **不实现** | 同上 |
| 28 | TG"积分换购"具体功能 | **不实现** | 同上 |
| 32 | 购买机器人 | **不实现** | NOTICE.md 已明确 |

---

## 4. Stage 0 残余清理点（已全部处理）

Stage 1 文档化过程中发现的、Stage 0 全局改名漏掉的点，登记在此，记录处理情况：

1. ✓ `install.sh` 第 4 行注释：`# X-Panel 统一安装脚本 (付费/免费二合一)` —— Stage 2 #5 已改为 `# x-panel-ce 安装脚本 (Community Edition, GPL-3.0)`。
2. ✓ `dnsjc.sh` 第 36 行 banner：`〔X-Panel-Pro 面板〕专属 "服务器 DNS 检测"` —— Stage 2 #9 已改为 `〔x-panel-ce〕服务器 DNS 检测`。
3. ✓ `x-ui.sh` 中 sublink 相关历史代码（约 80 行）+ banner "〔X-Panel 面板〕专属定制" 营销话术 —— Stage 0.5 收尾清理 commit `97c81632` 已整段删除，重写 `subconverter()` 为中性 OSS 提示函数（说明下线原因 + 引导查阅 ROADMAP）。
4. ✓ TG bot 抽奖框架整套（runLotteryDraw / sendLotteryGameInvitation / SendStickerToTgbot / 三处 callback / 主菜单"🎁 娱乐抽奖"按钮 / LotteryWin 数据表 / ceReportPrefs.Lottery 字段 / LOTTERY_STICKER_IDS 常量）—— Stage 0.5 收尾清理 commit `8c8d702e` 已整段移除（作为 Stage 0.1 隐私后门事件的延续治理）。详见 NOTICE.md 第 5 项。

---

## 5. 术语在 CE 中的对照

| 上游术语 | 含义 | CE 替代 |
|---|---|---|
| Pro 版 | 付费授权版本 | x-panel-ce（GPL-3.0 开源 fork） |
| 授权码 / 授权服务器 | 商业授权机制 | **不实现**；如保留 #17，则改为"部署 ID 上报（默认关闭）" |
| 会员等级 | 付费档位 | "角色"（admin / user） |
| 积分 / 签到 | 商业增值体系 | **不实现** |
| 购买机器人 | 销售渠道 | **不实现** |

---

## 6. 维护约定

- 每个 Stage 落地后，更新本文件对应行的状态（增加 `Done in commit <hash>` 列）。
- 任何对路线图编号 1-32 的实现都必须在 commit message 中显式引用编号（如 `feat(ce): #14 add 5 dashboard themes`）。
- 新增条目（即超出原 README 路线图的 CE 自有功能）从 100 起编号，避免与上游冲突。
- Release 命名约定：`ce-<MAJOR>.<MINOR>.<PATCH>`（与上游 X-Panel 的 `v*.*.*` 区分）。
- CE **不发布预构建 Docker 镜像**（QEMU 多平台 build 在公共 runner 上耗时 30-60 分钟、缓存命中率低、对 99% 走 systemd 直装的用户无收益）。仓库保留 `Dockerfile` 供有需要的贡献者本地构建。

---

## 7. Release 历史

| Release | Tag | 完成范围 | 说明 |
|---|---|---|---|
| 首个完整版 | [`ce-1.0.0`](https://github.com/hehelove/x-panel-ce/releases/tag/ce-1.0.0) | Stage 0 / 0.1 / 0.2 / 0.5 / 1 / 2 / 3 / 4 + Stage 5/6 决策化收尾 | 不实现 #26/#27/#28（积分） + #32（购买机器人）；CE 永久反遥测承诺 |
| Stage 7 易用性首发 | [`ce-1.1.0`](https://github.com/hehelove/x-panel-ce/releases/tag/ce-1.1.0) | #100（自动 DB 备份 + 保留 14 份） / #101（批量导出节点链接） / #102（批量启用/禁用/删除入站） | 升级 / 误删风险显著下降；常用运维操作从多点击折叠为一次性批量 |
| Stage 7 可观测性 | [`ce-1.2.0`](https://github.com/hehelove/x-panel-ce/releases/tag/ce-1.2.0) | #103（入站健康度三色圆点 + 30s 缓存轮询） / #104（TG `/health` 综合健康度报告） / #112（Cron 任务只读可视化） | 孤儿入站 / 端口监听异常 / cron 漂移立即可见；出门在外 TG 一键拉报告 |

---

## 8. 部署验证（交付物）

为了让用户/贡献者能在真实 VPS 上**可重现地**验证 ROADMAP 全部交付，
本仓库提供两份配套验证物：

| 文件 | 用途 |
|---|---|
| [`tools/ce-vps-smoke.sh`](../tools/ce-vps-smoke.sh) | 幂等只读自检脚本：12 个 section 覆盖 BBR / 二进制反检测扫描 / systemd / 端口 / 数据库 schema / HTTP 探测 / xray 子进程 / TG 反遥测采样 / 5 主题 |
| [`VPS-VERIFICATION-CHECKLIST.md`](./VPS-VERIFICATION-CHECKLIST.md) | 配套手动 checklist：脚本无法覆盖的 UI / 跨日 cron / 客户端协议握手 / 长尾运行时行为 |

VPS 上一行使用：

```bash
curl -fsSL https://raw.githubusercontent.com/hehelove/x-panel-ce/main/tools/ce-vps-smoke.sh | sudo bash
```

> 注意：当前 `release.yml` 中 `upload-release-action` 设了
> `event.action == 'published'` 限定，所以 push 触发的 workflow run **不会**
> 把 tar.gz 上传到 GitHub Release 页面。`install.sh` 一键安装前需要先在
> GitHub UI 上手动 publish 一次 `ce-1.0.0` Release（或者用源码 `go build`
> 部署）。详见 VPS-VERIFICATION-CHECKLIST.md §0.2。

---

最后更新：2026-04-26（追加 §8 部署验证交付物：ce-vps-smoke.sh + VPS-VERIFICATION-CHECKLIST.md）
