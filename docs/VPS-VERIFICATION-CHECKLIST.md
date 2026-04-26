# VPS 部署验证清单（VPS Verification Checklist）

> 本文档配套脚本 [`tools/ce-vps-smoke.sh`](../tools/ce-vps-smoke.sh) 使用，
> 给出 `x-panel-ce` 在真实 VPS 上的端到端验证步骤。
>
> 设计原则：脚本能验的事**不重复列在手动 checklist 里**；只列脚本无法
> 自动验证的项（UI 交互 / 跨日 cron / 客户端协议握手 / 长尾运行时行为）。

---

## 0. 准备工作

### 0.1 推荐 VPS 规格

| 项目 | 最低 | 建议 |
|---|---|---|
| OS | Ubuntu 20.04+ / Debian 11+ / CentOS 8+ | Debian 12 或 Ubuntu 22.04 |
| CPU | 1 core | 2 core |
| RAM | 512 MB | 1 GB |
| 磁盘 | 2 GB | 10 GB |
| 网络 | 解锁 22 / 80 / 443 / 面板端口 (默认 2053) | 同左 |
| 内核 | ≥ 4.9 | ≥ 5.4（BBR 默认可用） |
| 防火墙 | 放行 panel 端口 + 入站协议端口 | 同左 |

### 0.2 部署前置：GitHub Release Publish（一次性）

⚠️ **当前阻塞点**：本仓库 `release.yml` 中 `upload-release-action` 设了
`if: github.event_name == 'release' && github.event.action == 'published'`，
意味着 push 触发的 workflow run 不会把 `x-ui-linux-*.tar.gz` 上传到
**GitHub Release 页面**（只在 actions artifact 保留 90 天）。

后果：`install.sh` 调 `https://api.github.com/repos/hehelove/x-panel-ce/releases/latest`
和 `https://github.com/hehelove/x-panel-ce/releases/download/<tag>/x-ui-linux-*.tar.gz`
都会 **404**。

解决方法（任选）：

- **方法 A（推荐）**：去 GitHub UI 手动 publish 一次 `ce-1.0.0` Release：
  - 浏览器打开 https://github.com/hehelove/x-panel-ce/releases/new
  - `Choose a tag` 下拉选 `ce-1.0.0`
  - 标题填 `ce-1.0.0`
  - 描述用 `docs/ROADMAP.md` 的 §7 表格内容
  - 点 `Publish release`
  - 这会触发 `release.yml` 重跑一次（这次 `event.action == 'published'`），
    构建产物会自动上传到 release 页面，install.sh 即可下载

- **方法 B（VPS 部署方测试）**：跳过 install.sh，手动在 VPS 上 `git clone` +
  `go build`。详见 §1.C。

### 0.3 把脚本拷贝到 VPS

```bash
# 在 VPS 上：
curl -fsSL -o ce-vps-smoke.sh \
  https://raw.githubusercontent.com/hehelove/x-panel-ce/main/tools/ce-vps-smoke.sh
chmod +x ce-vps-smoke.sh
```

或者等部署完毕后从 `/usr/local/x-ui/` 找（如果你把它打进了 release tarball；
当前仓库里这个脚本**只在源码树**，未打包进 release，所以建议直接 curl）。

---

## 1. 部署方式（三选一）

### 1.A 一键安装（推荐，前提：§0.2 已 publish release）

```bash
# 安装最新版
bash <(curl -Ls https://raw.githubusercontent.com/hehelove/x-panel-ce/main/install.sh)

# 或安装指定版本
bash <(curl -Ls https://raw.githubusercontent.com/hehelove/x-panel-ce/main/install.sh) ce-1.0.0
```

完成后会输出登录账号 / 密码 / panel URL。

### 1.B Docker 部署

```bash
git clone https://github.com/hehelove/x-panel-ce.git
cd x-panel-ce
docker compose up -d --build
docker compose logs -f x-panel
```

⚠️ `docker-compose.yml` 用 `network_mode: host`，端口直接暴露在宿主机；
默认 panel 端口 2053。

### 1.C 源码编译（无依赖 release 页面）

```bash
# 装 go 1.26+
git clone https://github.com/hehelove/x-panel-ce.git /opt/x-panel-ce
cd /opt/x-panel-ce
CGO_ENABLED=1 go build -ldflags '-w -s' -o /usr/local/x-ui/x-ui main.go

# Xray 二进制需要单独下，参考 release.yml 的 Xray_URL 部分
mkdir -p /usr/local/x-ui/bin
# ... 下载 xray + geoip.dat / geosite.dat（详见 release.yml）

cp x-ui.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now x-ui
```

---

## 2. 自动验证：跑 `ce-vps-smoke.sh`

部署完毕后跑一遍：

```bash
# 默认：检查 binary + runtime
sudo bash ce-vps-smoke.sh

# 只查 binary，不查 runtime
bash ce-vps-smoke.sh --no-runtime

# 自定义二进制 / db 路径
sudo bash ce-vps-smoke.sh --bin /custom/x-ui --db /custom/x-ui.db

# 自定义面板端口（避免脚本读 setting）
sudo bash ce-vps-smoke.sh --port 12345
```

### 2.1 输出解读

| 状态 | 含义 | 处置 |
|---|---|---|
| `[PASS]` | 该项验证通过 | 无需关注 |
| `[WARN]` | 验证未通过但属"可能合理"的灰色情况（功能未启用、容器无 systemd 等） | 看上下文判断 |
| `[FAIL]` | 验证未通过且明确不应发生 | **必须排查** |
| `[INFO]` | 信息性输出，不计入评分 | 无需关注 |

### 2.2 脚本覆盖范围

| Section | 主题 | 对应 ROADMAP 编号 |
|---|---|---|
| 1 | 环境探测 + BBR 状态 | #29 |
| 2 | 二进制存在性 + 静态链接验证 | — |
| 3 | 反检测：二进制 strings 扫描（lottery / xeefei.com / 中央上报 marker） | Stage 0.1 + 抽奖清理 |
| 4 | 文件系统脚本残留扫描 | Stage 0.1 + 抽奖清理 |
| 5 | systemd 服务 + journal 错误统计 | — |
| 6 | 端口监听 | — |
| 7 | 数据库 schema（必有 + 必无） | 抽奖清理 |
| 8 | HTTP 接口探测 + 响应内容反检测 | — |
| 9 | xray 子进程 | — |
| 10 | TG outbound 流量采样反遥测 | Stage 0.1 |
| 11 | 5 主题文件验证 | #14 |

---

## 3. 手动验证：Stage 0 反检测（脚本无法覆盖项）

### 3.1 配置文件无上游 hardcoded chat_id

```bash
# 上游 commit 9c5599d2 之前曾硬编码上报到开发者控制的 chat_id；
# CE 必须无此残留。
sudo grep -rE '\-?[0-9]{9,}' /etc/x-ui/ 2>/dev/null | \
    grep -vE 'INTEGER|^$|inbounds|users' | head
```

**期待**：无任何形如 `-100xxxxxxxxxx` / `100xxxxxxxxx` 的 Telegram chat ID
（除非用户**自己**配置的 admin_chat_id —— 这是合法的）。

### 3.2 抽奖数据表不应被 AutoMigrate

```bash
sudo sqlite3 /etc/x-ui/x-ui.db '.tables' | tr ' ' '\n' | sort
```

**期待**：列表中**不应**出现 `lottery_wins`。
（如果是从旧版迁移过来的，可能残留这张表，可手动 `DROP TABLE lottery_wins;`，
但 CE 已不再 AutoMigrate，新装环境绝对不会有。）

---

## 4. 手动验证：Stage 2 安装脚本

### 4.1 #5 install.sh 跑通且无商业语

部署中观察 install.sh 输出：

- [ ] 无 "Pro 版" / "授权" / "激活码" / "支付" 等商业字眼
- [ ] 无"远程授权服务器"调用
- [ ] 头部 banner 显示 `〔x-panel-ce〕` 字样
- [ ] 末尾打印仓库地址为 `https://github.com/hehelove/x-panel-ce`
- [ ] License 显示 `GPL-3.0`

### 4.2 #7 webssh

```bash
systemctl status webssh
ss -tlnp | grep webssh   # 默认端口
```

- [ ] webssh systemd unit 已注册
- [ ] webssh 端口监听中（如已启用）

### 4.3 #8 线路 / IP 质量检测

进入 `x-ui` 菜单：

```bash
sudo x-ui
```

- [ ] 菜单中含 "线路检测" 或 "IP 质量" 选项
- [ ] 选项可触发对应脚本（`linejc.sh` / `dnsjc.sh`），输出 IP 地理 / 解锁信息

### 4.4 #29 内核调优 (BBR)

x-ui 菜单中跑 BBR 一键调优后：

```bash
sysctl net.ipv4.tcp_congestion_control      # 应为 bbr
sysctl net.core.default_qdisc               # 应为 fq
lsmod | grep bbr                            # 应有 tcp_bbr 模块
```

- [ ] 拥塞控制 = `bbr`
- [ ] 默认 qdisc = `fq`
- [ ] `tcp_bbr` 内核模块已加载

### 4.5 #21 / #22 SSL 证书

- [ ] x-ui 菜单中能找到 "SSL 证书申请" / "证书管理" 入口
- [ ] 可选择 acme.sh 一键申请（需要域名解析到 VPS）
- [ ] 申请完成后 `/root/cert/` 下有 `*.cer` + `*.key`

---

## 5. 手动验证：Stage 3 TG Bot（需先在面板里配置 TG bot token + admin chat_id）

### 5.1 #11 版本字串

发送 `/status` 给 bot：

- [ ] 回复中显示版本号（如 `x-panel-ce ce-1.0.0` 或 commit hash）
- [ ] 不显示上游 `xeefei` 字样

### 5.2 #16 每日报告 4 个 section

等到下次每日报告（或手动触发）：

- [ ] 报告含 4 个 section（**非 5 个** —— 抽奖 section 已删除）
- [ ] 4 个 section 应为：CPU/RAM 状态、流量统计、入站列表、面板版本（具体细节见
  `web/service/tgbot.go`）
- [ ] 报告**不含** "🎁 抽奖" / "lottery" / "中奖" 等字样

### 5.3 #25 主菜单按钮

发送 `/start`：

- [ ] 主菜单**不**包含 "🎁 娱乐抽奖" 按钮
- [ ] 主菜单**不**包含任何与抽奖 / 中奖 / 兑换相关的按钮

### 5.4 反遥测：观察 30 分钟无非用户配置的 outbound

用 ce-vps-smoke.sh 跑 `--tg-sec 1800`（30 分钟采样）：

```bash
sudo bash ce-vps-smoke.sh --tg-sec 1800
```

或者手动用 `tcpdump`：

```bash
sudo tcpdump -n -i any 'host 149.154.160.0/20 or host 91.108.4.0/22' -c 100
```

- [ ] 30 分钟内只有发往**你自己配置的 admin_chat_id 所在的 TG 数据中心**的连接
- [ ] 无任何到 `t.me/bot...` / `api.telegram.org` 的"额外"连接（即非主动用户操作触发的）

---

## 6. 手动验证：Stage 4 面板 UI

需要浏览器登录面板 (`https://VPS_IP:PORT/PATH`)：

### 6.1 #1 + #24 品牌

- [ ] 登录页标题为 `x-panel-ce` 或 `X-Panel-CE`，**不含** `xeefei` / 商业 LOGO
- [ ] 浏览器标签 favicon 不是上游商业 logo
- [ ] 关于 / 设置页底部显示 `GPL-3.0` 许可证 + 仓库链接 `hehelove/x-panel-ce`

### 6.2 #14 五主题切换

设置 / 主题区：

- [ ] 主题选项 ≥ 5（典型：cyan / dark / blue / purple / orange）
- [ ] 切换主题后页面颜色立即生效
- [ ] 刷新浏览器，主题保持（验证 localStorage 持久化）
- [ ] 退出登录再登入，主题保持

### 6.3 #3 Reality 随机 SNI

新建/编辑 inbound → 选 VLESS + Reality：

- [ ] SNI 字段旁有 "随机 SNI" / "推荐列表" 按钮
- [ ] 点击会从内置白名单中选一个常见 CDN 域名（如 `www.cloudflare.com` /
  `www.microsoft.com`）
- [ ] X25519 公钥 / 私钥旁有 "随机生成" 按钮，点击调 `xray x25519` 生成新对
- [ ] ShortIDs 旁有 "随机生成" 按钮

### 6.4 #13 + #31 一键配置 / 批量部署

入站列表页：

- [ ] 顶部有 "一键创建推荐入站" 按钮（典型为 VLESS+Vision+Reality）
- [ ] 顶部或工具栏有 "批量部署" / "批量创建" 按钮
- [ ] 批量创建可指定数量 + 端口范围 + 协议模板
- [ ] 创建中如果某个失败，**前面成功的也会被删除**（compensating rollback —— 验证：
  故意指定一个被占用端口在批次中部，看其他 inbound 是否最终都被清理）

### 6.5 #15 + #30 流量周期重置

新建/编辑 inbound：

- [ ] 客户端配置中含 "重置周期"（如：每日 / 每周 / 每月固定日 / 自定义周期）
- [ ] 选 "每月 X 日" 时可选 1-31 范围
- [ ] 设置后等到次日（或修改 cron 提前），观察 `total` / `used_traffic` 字段是否归零

⚠️ 这个 cron 调度由 `github.com/robfig/cron/v3` 在面板进程内注册，**不在系统 crontab**，
所以 `crontab -l` 看不到。验证方式：

```bash
# 留意 panel 进程日志
sudo journalctl -u x-ui --since today | grep -iE 'reset|traffic|cron'
```

---

## 7. 跑完之后

### 7.1 报告位置

`ce-vps-smoke.sh` 的报告写到 `/tmp/x-panel-ce-smoke-<时间戳>.log`，
里面包含全部 PASS/WARN/FAIL 详细信息 + 命中字串前 3 行（用于排查）。

### 7.2 提交反馈

如果出现 `[FAIL]`：

- 先在仓库 [Issues](https://github.com/hehelove/x-panel-ce/issues) 搜一下相同问题
- 没有就开新 issue，附上：
  - VPS smoke 报告（注意先 `sed` 脱敏 IP / hostname / 用户名 / token）
  - 部署方式（A/B/C）
  - install.sh 完整输出（如果是 install.sh 失败）
  - `journalctl -u x-ui --no-pager -n 200` 最近日志

### 7.3 脱敏建议

报告里可能含 IP / hostname / 端口，发到公网前建议：

```bash
sed -i \
  -e "s/$(hostname)/HOSTNAME/g" \
  -e 's/[0-9]\{1,3\}\.[0-9]\{1,3\}\.[0-9]\{1,3\}\.[0-9]\{1,3\}/X.X.X.X/g' \
  /tmp/x-panel-ce-smoke-*.log
```

---

## 8. 已知限制（脚本无法覆盖）

`ce-vps-smoke.sh` **不能**做下列事，必须手动验证：

| 不能验证项 | 原因 | 替代方案 |
|---|---|---|
| Reality 协议是否能从客户端真实连通 | 需要真实 v2rayN/sing-box 客户端配合 | 用客户端测速 + 抓包看 TLS handshake |
| 流量周期重置是否在午夜真实触发 | 验证窗口 < 24h | 等一天/手改面板 cron 表达式提前到 5 分钟后 |
| UI 主题切换体感 | 无浏览器 | 人眼比对 / Playwright 自动化（未来路线图） |
| 批量部署 compensating rollback | 需要构造端口冲突场景 | 手工：在批次中部塞个被占端口 |
| TG bot 每日报告内容 | 需要等到次日发送时间 | 手动调用 `/report now` 或类似命令 |
| BBR 真实加速效果 | 跨网络环境差异极大 | iperf3 对照测试 |

---

## 9. 与 ROADMAP 的对应关系

本 checklist 验证范围对应 [`docs/ROADMAP.md`](./ROADMAP.md) 的：

- **Stage 0**：§3.0 + 反检测扫描（脚本 Section 3、4、10）
- **Stage 0.5**：抽奖框架彻底删除（脚本 Section 3、4、7、8 + 手动 §3.2 / §5.2 / §5.3）
- **Stage 2**：§3.1 全部 7 个 item（手动 §4）
- **Stage 3**：§3.2 全部 9 个 item（手动 §5）
- **Stage 4**：§3.3 全部 8 个 item（手动 §6 + 脚本 Section 11）
- **Stage 5/6**：决策化收尾，无运行时验证项

完成本 checklist 即视为 **ROADMAP 全部交付物在真实环境通过验证**。
