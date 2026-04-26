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

| Stage | 主题 | 路线图编号 | 风险面 | 依赖前置 |
|---|---|---|---|---|
| **0** | 仓库合规化（已完成本地提交，待 push） | — | 极低 | — |
| **1** | 路线图文档化（本文件） | — | 零运行时 | Stage 0 |
| **2** | 安装脚本 CE 化 | 5 / 7 / 8 / 9 / 21 / 22 / 29 | 低（仅 shell + 第三方工具集成） | Stage 1 |
| **3** | TG Bot 通知与状态显示 | 2 / 4* / 6 / 10 / 11 / 12 / 16 / 19 / 25 | 中（涉及在线消息推送、定时任务） | Stage 2 |
| **4** | 面板后台 UI / 入站增强 | 1 / 3 / 13 / 14 / 15 / 24* / 30 / 31 | 中（前后端联动 + 数据库迁移） | Stage 3 |
| **5** | 主从与中转高级功能 | 18 / 20 / 23 | 高（远程操控被控端、数据库快照） | Stage 4 |
| **6** | 授权验证机制改造 | 17* | 待决策（删除 / 重定义为 CE 部署自检） | 全部前置完成后单独评审 |

> 标注 `*` 的条目在 CE 中需要**重定义而不是直接照搬**，详见下方明细。

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
| 21 | 证书申请"备用方式" | `x-ui.sh:ssl_cert_issue_main` 子菜单内增项 | acme.sh / certbot | OSS 重写 | 实现：HTTP-01 standalone（默认）+ DNS-01 manual 备用（用户手工填 TXT 后回车）；不依赖商业 DNS API。 |
| 22 | 自定义证书路径 | `x-ui.sh:ssl_cert_issue_main` 子菜单内增项 | 无 | OSS 重写 | 用户输入 fullchain/key 路径 → 校验存在性 + 权限 → 软链到 `/etc/x-ui/cert/`；无网络请求。 |
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

### 3.4 Stage 5 — 主从与中转高级

| # | 标题摘要 | 上游源码坐标 | 依赖评估 | CE 策略 | 备注 |
|---|---|---|---|---|---|
| 18 | TG"多面板管理"（一个 bot 控多 VPS） | `web/service/tgbot.go` + 新增 `master_slave` 数据模型 | 上游：可能有私有协议；CE 自定义 | OSS 重写 | 通信走面板自身现有 HTTP API + token，避免引入新协议 |
| 20 | "一键部署中转节点"（远端 Socks5 → 本机路由 → 二维码） | 多文件联动（前端 + tgbot + xray 模板） | 依赖 #18 主从就绪 | OSS 重写 | 风险高：跨机操作；分两步走，先完成单机配置生成，再做远程下发 |
| 23 | 数据快照 + 远程急救还原 | 新增 `web/service/snapshot.go` + CLI 子命令 | 无 | OSS 重写 | 快照内容：`/etc/x-ui/x-ui.db` + 关键 sysctl + xray 配置；用 tar.gz + sha256 校验 |

### 3.5 Stage 6 — 授权验证机制（专项决策）

| # | 标题摘要 | 上游源码坐标 | 依赖评估 | CE 策略 | 备注 |
|---|---|---|---|---|---|
| 17* | 授权码"后台联网验证"+ 机器指纹 | `install.sh`（已删 HWID）+ 上游远程授权服务器 | **强依赖闭源 Pro 服务器** | **待决策（默认删除）** | CE 不收费，因此该机制本质失效。Stage 6 单独评审：A 选项=完全移除；B 选项=保留"部署 ID + 版本上报"作为开源化匿名遥测（默认 opt-out） |

### 3.6 明确不实现

| # | 标题摘要 | CE 策略 | 备注 |
|---|---|---|---|
| 26 | 面板"签到得积分" | **不实现** | 用户明确要求跳过 |
| 27 | TG 签到积分 / 查询 / 换购 / 排行榜 | **不实现** | 同上 |
| 28 | TG"积分换购"具体功能 | **不实现** | 同上 |
| 32 | 购买机器人 | **不实现** | NOTICE.md 已明确 |

---

## 4. Stage 0 残余清理点（移交 Stage 2 顺手处理）

Stage 1 文档化过程中发现的、Stage 0 全局改名漏掉的点，登记在此，避免遗忘：

1. `install.sh` 第 4 行注释：`# X-Panel 统一安装脚本 (付费/免费二合一)` —— 已不符合 CE 单一开源路径，Stage 2 #5 顺手改为 `# x-panel-ce 安装脚本 (Community Edition, GPL-3.0)`。
2. `dnsjc.sh` 第 36 行 banner：`〔X-Panel-Pro 面板〕专属 "服务器 DNS 检测"` —— Stage 2 #9 顺手改为 `〔x-panel-ce〕服务器 DNS 检测`。
3. `x-ui.sh` 中 sublink 相关历史代码（第 1640-1675 行附近）虽然已被 Stage 0 的 CE 提示阻断，但仍以"参考代码"形式保留，Stage 2 评审是否一并删除。

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

---

最后更新：2026-04-26
