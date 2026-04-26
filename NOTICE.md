# NOTICE

本项目 **x-panel-ce**（X-Panel Community Edition）是基于
**[xeefei/X-Panel](https://github.com/xeefei/X-Panel)**（下称"上游 X-Panel"）的开源 fork。

---

## 1. 命名与性质

- 本仓库地址：<https://github.com/hehelove/x-panel-ce>
- 本仓库定位：**上游 X-Panel 的纯开源社区分支（Community Edition, CE）**。
- 本仓库**不**与上游"X-Panel Pro"或"付费 Pro 版"存在任何关联，**不**销售授权码，**不**接受赞助代币地址，**不**绑定任何机器人或第三方收款渠道。
- 上游中所有冠以"Pro"或与"购买授权码"绑定的功能，在本项目中将以**完全开源**的形式重写实现。

## 2. 许可证（License）

- 上游 X-Panel 采用 **GNU General Public License v3.0 (GPL-3.0)**。
- 本 fork 同样以 **GPL-3.0** 发布，[`LICENSE`](./LICENSE) 文件保持不变。
- 根据 GPL-3.0：
  - 您有权自由地使用、修改、分发本软件源代码。
  - 任何基于本项目的二次分发，必须同样以 GPL-3.0 发布并保留版权声明。
  - 上游版权归原作者所有；本 fork 在此基础上的修改归各贡献者所有。

## 3. 上游致谢（Upstream Credits）

本项目的代码基础源自以下作者与项目，特此致谢：

- **xeefei** — 上游 X-Panel 主要维护者：<https://github.com/xeefei/X-Panel>
- **MHSanaei** — 3X-UI 主要作者：<https://github.com/MHSanaei/>
- **alireza0** — alireza-xui 维护者：<https://github.com/alireza0/>
- **FranzKafkaYu** — x-ui 维护者：<https://github.com/FranzKafkaYu/>
- **vaxilu** — 早期 x-ui 作者：<https://github.com/vaxilu/>
- **Xray-core 项目组**：<https://github.com/XTLS/Xray-core>

## 4. 与上游的差异（Diff Summary）

> 本节给出**已经发生**的清理。CE 计划中"将以开源方式重写"的功能逐项明细
> （含上游源码坐标、依赖评估、CE 策略、所属 Stage），见
> [`docs/ROADMAP.md`](./docs/ROADMAP.md)。

为符合 fork 的合规与中立性原则，本项目相对上游做了如下清理：

1. **移除商业授权机制**
   - 删除 `install.sh` 中的硬件指纹（HWID）采集逻辑。
   - 删除 `install.sh` 中的 `install_paid_version()` 函数与远程授权服务器调用。
   - 删除安装脚本与面板对外引用的"购买授权码机器人"链接。
2. **移除赞助/广告链接**
   - 删除 README 中的 USDT 钱包地址、Buy Me a Coffee 入口、`xeefei.blogspot.com` 教程链接、若干 VPS 联盟营销（Affiliate）链接。
3. **替换仓库与镜像引用**
   - 全局将 `github.com/xeefei/x-panel` / `xeefei/X-Panel` 替换为 `github.com/hehelove/x-panel-ce`。
   - GHCR 镜像由 `ghcr.io/xeefei/x-panel` 调整为 `ghcr.io/hehelove/x-panel-ce`。
4. **保留兼容性**
   - 二进制名、systemd 服务名、CLI 命令名仍为 `x-ui`，数据库路径仍为 `/etc/x-ui/x-ui.db`，便于从上游 X-Panel / 3X-UI 平滑迁移。
   - Go module 名仍为 `x-ui`。
5. **【Stage 0.1 紧急安全清理】移除上游硬编码的 Telegram"中央统计"上报后门**
   - 上游 `web/service/tgbot.go` 中以常量形式硬编码了一个第三方 Telegram Bot Token
     (`REPORT_BOT_TOKEN`) 与三个上游开发者控制的频道 ID (`REPORT_CHAT_IDS`)。
   - 三处异步 goroutine 会在以下时机自动把用户信息上传到该中央频道：
     - `SendReport()` 每次执行时（主机名 + 时间戳，"心跳报告"）；
     - 抽奖回调中奖时（TG 用户名 + TG 用户 ID + 主机名，"中奖报告"）；
     - 抽奖回调未中奖时（TG 用户名 + TG 用户 ID + 主机名，"未中奖报告"）。
   - 这违反 GPL-3.0 软件最基本的用户隐私底线，且部署用户毫无知情或退出途径。
   - **CE 已在 Stage 0.1 中将常量与三处上报代码整段移除**；TG 报告功能现在仅向
     当前部署用户自己配置的管理员 chat 发送。任何下游 fork 不应再恢复该上报路径。
   - **【后续清理】抽奖功能整套移除**：作为 Stage 0.1 的延续治理，抽奖框架本身
     （`runLotteryDraw` / `sendLotteryGameInvitation` / 三处 callback / 主菜单
     "🎁 娱乐抽奖"按钮 / `LotteryWin` 数据表 / `ceReportPrefs.Lottery` 字段 /
     `LOTTERY_STICKER_IDS` 常量 / `SendStickerToTgbot` 辅助方法）也已在 Stage 4
     收尾清理中整段移除：(1) 与 CE 开源、自托管、无收款定位无关；(2) 留着抽奖
     框架等于给未来恢复中央上报路径留接口。`lottery_wins` 表仅在历史部署中
     存在，CE 不再 AutoMigrate；如需彻底清理表，可在迁移到 CE 后于面板 SQLite
     中手动 `DROP TABLE lottery_wins;`。
6. **【Stage 6 决策】拒绝任何形式的"匿名遥测 / 部署 ID 上报"**
   - 上游"授权码后台联网验证 + 机器指纹"机制（路线图 #17）在 CE 中已无运行入口。
   - CE **永久承诺**：面板与 TG bot 在任何配置下，**只**与用户在本机配置的
     Xray / 数据库 / Telegram 管理员 chat 通信，**不**会向 hehelove 或任何第三方
     聚合服务器发送任何数据（包括但不限于：版本号、部署 ID、流量统计、用户
     标识、主机指纹）。
   - 该决策与 GPL-3.0 + 用户自托管定位一致，且作为 Stage 0.1 隐私后门事件的延
     续治理，写入 `docs/ROADMAP.md § Stage 6` 作为长期约束。下游 fork 如需恢复
     遥测路径，必须以**显著的 README 章节 + 默认关闭的开关**形式公开告知。

## 5. 用户责任声明

- 本项目仅供学习与研究 Xray / VLESS / Reality 等代理协议之用。
- 使用者应自行遵守所在国家或地区的法律法规，**不得**用于任何非法用途。
- 因使用本项目造成的任何后果，由使用者自行承担，与本项目维护者及上游作者均无关。

## 6. 反馈与贡献

- Issues / Pull Requests：<https://github.com/hehelove/x-panel-ce/issues>
- 本项目接受任何符合 GPL-3.0 条款的开源贡献。

## 7. 不发布预构建 Docker 镜像（运营决策）

- ce-1.0.2 起，CE **不再发布预构建 Docker 镜像**到 `ghcr.io`。
- 原因：5 平台 QEMU 多架构 build 在 GitHub Actions 公共 runner 上耗时
  30-60 分钟/次、缓存命中率低、外部依赖 (`Loyalsoldier/v2ray-rules-dat`、
  `chocolate4u/Iran-v2ray-rules` 等) 网络抖动经常 flake，对 99% 走
  systemd 直装的用户无实际收益。
- 仓库保留 `Dockerfile` + `DockerInit.sh` + `DockerEntrypoint.sh`，需要
  docker 部署的贡献者请按 README 「Docker 安装（本地构建）」章节自行
  `docker build -t x-panel-ce:local .` 即可。
- 已删除 `.github/workflows/docker.yml`，避免每次 release 触发空跑/失败
  CI 浪费用户/maintainer 心智成本。

---

最后更新：2026-04-26（追加 Stage 6 反遥测承诺；追加抽奖功能整套移除说明；追加 §7 不发布预构建 Docker 镜像决策）
