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

## 5. 用户责任声明

- 本项目仅供学习与研究 Xray / VLESS / Reality 等代理协议之用。
- 使用者应自行遵守所在国家或地区的法律法规，**不得**用于任何非法用途。
- 因使用本项目造成的任何后果，由使用者自行承担，与本项目维护者及上游作者均无关。

## 6. 反馈与贡献

- Issues / Pull Requests：<https://github.com/hehelove/x-panel-ce/issues>
- 本项目接受任何符合 GPL-3.0 条款的开源贡献。

---

最后更新：2026-04-26
