<!--
================================================================================
  x-panel-ce — Community Edition fork of xeefei/X-Panel
  本仓库为 hehelove/x-panel-ce，是上游 xeefei/X-Panel 的纯开源 fork，
  与上游 "X-Panel Pro" / "付费 Pro 版" 无关，所有 Pro 功能将以开源方式重写。
  详细信息见 NOTICE.md 与 LICENSE（GPL-3.0）。
================================================================================
-->

# x-panel-ce

[![](https://img.shields.io/github/v/release/hehelove/x-panel-ce.svg?style=for-the-badge)](https://github.com/hehelove/x-panel-ce/releases)
[![](https://img.shields.io/github/actions/workflow/status/hehelove/x-panel-ce/release.yml.svg?style=for-the-badge)](https://github.com/hehelove/x-panel-ce/actions)
[![GO Version](https://img.shields.io/github/go-mod/go-version/hehelove/x-panel-ce.svg?style=for-the-badge)](#)
[![Downloads](https://img.shields.io/github/downloads/hehelove/x-panel-ce/total.svg?style=for-the-badge)](https://github.com/hehelove/x-panel-ce/releases/latest)
[![License](https://img.shields.io/badge/license-GPL%20V3-blue.svg?longCache=true&style=for-the-badge)](https://www.gnu.org/licenses/gpl-3.0.en.html)

> **关于本仓库 / About this repository**
>
> 本仓库 `hehelove/x-panel-ce` 是上游 [`xeefei/X-Panel`](https://github.com/xeefei/X-Panel) 的**纯开源社区分支（Community Edition, CE）**。本 fork 与上游"X-Panel Pro 付费版"**无任何关联**：
> - **不**销售授权码，**不**接受赞助代币地址，**不**绑定任何"购买机器人"。
> - **不**进行任何匿名遥测或部署 ID 上报。
> - 上游所有冠以"Pro"的功能，均在本 CE 中以**完全开源**的形式重写。
>
> 详见 [`NOTICE.md`](./NOTICE.md) 与 [`docs/ROADMAP.md`](./docs/ROADMAP.md)；许可证保持 [GPL-3.0](./LICENSE)。

> **声明**：本项目仅供个人学习、研究 Xray / VLESS / Reality 等代理协议之用。请遵守当地法律法规，**不得**用于任何非法用途，亦不建议用于生产环境。因使用本项目造成的任何后果由使用者自行承担。

---

## 项目简介

`x-panel-ce` 是基于 Xray-core 的多协议代理面板，提供 Web UI 与 Telegram 机器人两套管理入口。

支持协议：VMess、VLESS、Trojan、Shadowsocks、Dokodemo-door、Socks、HTTP、WireGuard，以及 XTLS 原生协议族（RPRX-Direct、Vision、REALITY）。

---

## 功能特性

### 上游既有

- 系统状态查看与监控
- 多用户、多协议、多入站
- 流量统计、流量限制、过期时间限制
- 可搜索所有入站和客户端信息
- 深色 / 浅色主题切换
- 一键 SSL 证书申请与自动续签（ACME / Cloudflare / Certbot）
- 自定义 Xray 配置模板
- 支持从面板导出 / 导入数据库
- HTTPS 访问面板（自备域名 + SSL 证书）
- IP 限制（基于 fail2ban）
- WARP 路由（v2.1.0+ 内置）
- Telegram 机器人通知与远程管理

### CE 新增 / 重写（开源方式）

完整路线图与每条任务的实现细节见 [`docs/ROADMAP.md`](./docs/ROADMAP.md)。本次发布已落地的核心条目：

| 类别 | 已实现功能 |
|---|---|
| 安装脚本 | webssh 一键部署、线路 / IP 质量检测、地区 DNS 检测、内核深度调优（BBR + FQ + TCP Fast Open + 缓冲区，支持 dry-run / backup / apply / rollback） |
| 面板 UI | 5 套配色主题（标准 / 炫彩 / 深海 / 暮光 / 幽林），首选项 `localStorage` 持久化 |
| Reality | "随机偷域"按钮（内置 10 个候选 SNI 候选池） |
| 流量管理 | 客户端流量自动周期重置（每日 / 每周 / 每月任意 1–31 号，月底 fallback） |
| 批量部署 | 一键批量生成 VLESS + TCP + Reality + Vision 入站（端口冲突检测 + 失败补偿回滚） |
| Telegram 机器人 | `/checkupdate` 读取 CE Release Notes、`/selfcheck` 部署自检、`/getlinks` 列出本机入站节点、`/webssh` 对接 webssh 服务、入站上 / 下线通知钩子、每日报告内容可定制（开关 / 时间 / 频率） |
| 安全治理 | 移除上游硬编码 Telegram 中央上报后门、移除商业授权机制（HWID 采集 / 远程授权服务器调用 / 购买入口） |

> 路线图中归属 Stage 5（多面板管理 / 一键中转部署 / 数据快照）的三项跨机功能为**长期项目**，需要新建 Server 表与 SSH 远程执行框架，CE 当前未立项；前端入口保留并显示中性提示。
> 路线图中 Stage 6（授权验证）已**完全移除**，CE 永久承诺零遥测。

---

## 快速安装

```sh
bash <(curl -Ls https://raw.githubusercontent.com/hehelove/x-panel-ce/master/install.sh)
```

如果系统未安装 `curl`：

```sh
apt update -y && apt install -y curl socat
```

### 安装指定版本

```sh
VERSION=v26.2.15 && bash <(curl -Ls "https://raw.githubusercontent.com/hehelove/x-panel-ce/$VERSION/install.sh") $VERSION
```

### 升级

进入 `x-ui` 命令行菜单，选择 **2. 更新面板**。如需保留旧设置，提示时输入 `n`。

### 从其他 x-ui 分支迁移

`x-panel-ce` 与上游 X-Panel / 3X-UI 数据库路径一致（`/etc/x-ui/x-ui.db`），通常可直接覆盖安装；若使用 F 佬 `x-ui` 或更早分支，请先备份数据库再安装。

---

## 默认面板设置

| 项 | 值 |
|---|---|
| 端口 | `13688`（可在面板内修改） |
| 用户名 / 密码 / 访问路径 | 默认安装时随机生成（也可在安装时自定义） |
| 数据库路径 | `/etc/x-ui/x-ui.db` |
| Xray 配置路径 | `/usr/local/x-ui/bin/config.json` |
| 面板访问 URL（HTTPS） | `https://你的域名:13688/访问路径/panel` |

---

## 安全建议

1. **不要**用 `http://` 明文模式登录面板，明文会造成密码与节点信息泄露。
2. 二选一加密登录方式：
   - **推荐**：申请 SSL 证书（脚本菜单 18），用 `https://域名/路径/panel` 登录。
   - **备选**：本地 `ssh -L` 端口转发，再用 `http://127.0.0.1:本地端口/路径/panel` 登录。
3. 首次登录后修改默认用户名 / 密码 / 访问路径。
4. 创建入站时使用高位端口（建议 `40000–65000`），避免低位常见端口被探测。
5. 同 IP + 同端口不要在多省份 / 多终端同时使用，避免被 GFW 视为机场流量特征。

---

## SSL 证书

进入 `x-ui` 命令行菜单，选择 **18. 申请 SSL 证书**：

- 子选项 1：常规 ACME 模式（HTTP-01，需 80 端口放行）
- 子选项 5：备用模式（常规失败时尝试）
- 子菜单包含：获取证书 / 吊销证书 / 续签证书 / 显示所有证书 / 设置面板证书路径
- 支持 Cloudflare DNS-01（需 CF 邮箱 + Global API Key）
- 自定义证书路径：手动上传到 VPS 后填路径

证书默认每 3 个月自动续签一次，需保证 80 端口持续放行。

---

## Telegram 机器人

通过 `@BotFather` 创建机器人获得 Token，然后在面板 → 设置 → Telegram 机器人配置中填入：

| 字段 | 说明 |
|---|---|
| Token | `@BotFather` `/newbot` 返回的字符串 |
| Admin Chat ID | 自己的 TG 用户 ID（向 [@useridinfobot](https://t.me/useridinfobot) 发任意消息可获取） |
| 通知 Cron | 标准 cron 表达式（如 `@daily`、`@hourly`、`30 * * * * *`） |

### 机器人功能

- 定期报告（流量 / CPU / 阈值 / 到期时间）
- 登录通知
- 数据库备份（自动 + 按需 `/createbackup`）
- 客户端报告菜单（按 UUID / 密码 / Email 查询）
- `/checkupdate` 查看 CE 最新版本与变更
- `/selfcheck` 部署自检（无需联网授权服务器）
- `/getlinks` 列出本机所有入站的节点链接
- `/webssh` 对接面板 webssh 服务
- 入站上 / 下线通知钩子（拼车场景明确知道节点状态）
- 多语言菜单

---

## 操作系统支持

Ubuntu 20.04+、Debian 11+、CentOS 8+、AlmaLinux 8+、Rocky Linux 8+、Oracle Linux 8+、Fedora 36+、OpenEuler 22.03+、Arch Linux、Manjaro、Armbian、OpenSUSE Tumbleweed、Amazon Linux 2023。

## 架构支持

`amd64` / `x86 / i386` / `arm64 / armv8 / aarch64` / `armv7` / `armv6` / `armv5` / `s390x`

二进制由 GitHub Actions 自动构建并发布到 Releases 与 GHCR。

---

## 多语言支持

英语、波斯语、简体中文、繁体中文、俄语、越南语、西班牙语、印尼语、乌克兰语、土耳其语、葡萄牙语。

进入面板登录页或后台 → 设置切换语言。Telegram 机器人语言独立设置（面板设置 → 机器人配置）。

---

## API

`POST /login` 用 `{username, password}` 登录后，所有入站操作都在 `/panel/api/inbounds/` 下。完整路由表（节选）：

| 方法 | 路径 | 操作 |
|---|---|---|
| `GET` | `/list` | 获取所有入站 |
| `GET` | `/get/:id` | 获取单条入站 |
| `GET` | `/getClientTraffics/:email` | 按 email 查询客户端流量 |
| `GET` | `/getClientTrafficsById/:id` | 按 ID 查询客户端流量 |
| `GET` | `/createbackup` | 让 TG bot 给管理员发备份 |
| `POST` | `/add` | 添加入站 |
| `POST` | `/del/:id` | 删除入站 |
| `POST` | `/update/:id` | 更新入站 |
| `POST` | `/addClient` | 给入站添加客户端 |
| `POST` | `/:id/delClient/:clientId` | 删除客户端 |
| `POST` | `/updateClient/:clientId` | 更新客户端 |
| `POST` | `/:id/resetClientTraffic/:email` | 重置单客户端流量 |
| `POST` | `/resetAllTraffics` | 重置所有入站流量 |
| `POST` | `/resetAllClientTraffics/:id` | 重置入站下所有客户端流量 |
| `POST` | `/delDepletedClients/:id` | 删除入站耗尽的客户端（`-1` = all） |
| `POST` | `/onlines` | 获取在线 email 列表 |
| `POST` | `/ce/quickDeployReality` | **CE 新增**：批量部署 Reality 入站 |

`clientId` 字段：VMESS / VLESS 用 `client.id`，TROJAN 用 `client.password`，Shadowsocks 用 `client.email`。

---

## 环境变量

| 变量 | 类型 | 默认值 |
|---|---|---|
| `XUI_LOG_LEVEL` | `debug` / `info` / `warn` / `error` | `info` |
| `XUI_DEBUG` | `boolean` | `false` |
| `XUI_BIN_FOLDER` | `string` | `bin` |
| `XUI_DB_FOLDER` | `string` | `/etc/x-ui` |
| `XUI_LOG_FOLDER` | `string` | `/var/log` |

示例：

```sh
XUI_BIN_FOLDER="bin" XUI_DB_FOLDER="/etc/x-ui" go build main.go
```

---

## Docker 安装

<details>
<summary>展开</summary>

```sh
bash <(curl -sSL https://get.docker.com)

git clone https://github.com/hehelove/x-panel-ce.git
cd x-panel-ce
docker compose up -d
```

或不用 compose：

```sh
docker run -itd \
   -e XRAY_VMESS_AEAD_FORCED=false \
   -v $PWD/db/:/etc/x-ui/ \
   -v $PWD/cert/:/root/cert/ \
   --network=host \
   --restart=unless-stopped \
   --name x-panel-ce \
   ghcr.io/hehelove/x-panel-ce:latest
```

升级：

```sh
cd x-panel-ce
docker compose down
docker compose pull
docker compose up -d
```

卸载容器：

```sh
docker stop x-panel-ce && docker rm x-panel-ce
```

</details>

---

## 手动安装

<details>
<summary>展开</summary>

```sh
ARCH=$(uname -m)
case "${ARCH}" in
  x86_64 | x64 | amd64) XUI_ARCH="amd64" ;;
  i*86 | x86) XUI_ARCH="386" ;;
  armv8* | armv8 | arm64 | aarch64) XUI_ARCH="arm64" ;;
  armv7* | armv7) XUI_ARCH="armv7" ;;
  armv6* | armv6) XUI_ARCH="armv6" ;;
  armv5* | armv5) XUI_ARCH="armv5" ;;
  s390x) echo 's390x' ;;
  *) XUI_ARCH="amd64" ;;
esac

wget https://github.com/hehelove/x-panel-ce/releases/latest/download/x-ui-linux-${XUI_ARCH}.tar.gz

cd /root/
rm -rf x-ui/ /usr/local/x-ui/ /usr/bin/x-ui
tar zxvf x-ui-linux-${XUI_ARCH}.tar.gz
chmod +x x-ui/x-ui x-ui/bin/xray-linux-* x-ui/x-ui.sh
cp x-ui/x-ui.sh /usr/bin/x-ui
cp -f x-ui/x-ui.service /etc/systemd/system/
mv x-ui/ /usr/local/
systemctl daemon-reload
systemctl enable x-ui
systemctl restart x-ui
```

</details>

---

## 备份与恢复

通过面板设置好 Telegram 机器人，可点击对应菜单获取 `x-ui.db` 与 `config.json` 两个备份文件。在新 VPS 安装本面板后，将这两个文件分别覆盖到 `/etc/x-ui/x-ui.db` 与 `/usr/local/x-ui/bin/config.json`，重启面板即可完成迁移。证书路径如果域名一致通常无需调整。

---

## 防火墙

新装系统默认会拦截入站端口。请放行：

- 面板登录端口（默认 `13688`，可改）
- 出 / 入站协议使用的端口
- 申请 SSL 证书 + 自动续签：必须放行 `80` 与 `443`

`x-ui` 命令行菜单第 21 选项可一键安装防火墙规则模板（`ufw` / `firewalld`）。

---

## 反馈与贡献

- Issues / Pull Requests：<https://github.com/hehelove/x-panel-ce/issues>
- 本项目接受任何符合 GPL-3.0 条款的开源贡献。

---

## 致谢

本仓库代码以 [GPL-3.0](./LICENSE) 协议发布。在此向以下上游项目维护者致谢，他们的工作是 x-panel-ce 的基础：

- [xeefei](https://github.com/xeefei/) — 上游 X-Panel 主要维护者
- [MHSanaei](https://github.com/MHSanaei/) — 3x-ui 维护者
- [alireza0](https://github.com/alireza0/)
- [FranzKafkaYu](https://github.com/FranzKafkaYu/)
- [vaxilu](https://github.com/vaxilu/) — 早期 x-ui 项目

路由规则数据集致谢：

- [Iran v2ray rules](https://github.com/chocolate4u/Iran-v2ray-rules)（GPL-3.0）
- [Vietnam Adblock rules](https://github.com/vuong2023/vn-v2ray-rules)（GPL-3.0）

---

## Star 趋势

[![Stargazers over time](https://starchart.cc/hehelove/x-panel-ce.svg)](https://starchart.cc/hehelove/x-panel-ce)
