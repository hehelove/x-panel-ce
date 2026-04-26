# Stage 7 — 易用性 / 可观测性 / 运维增强 实施规划

> **For Claude:** 执行本计划时，每完成一个编号项即提交一次（commit message 引用编号 #N），并按 release 切分推 tag。每项都要在动手前用 [`writing-plans`](../../../.cursor/skills/writing-plans/SKILL.md) 风格再产出一份子计划。

**目标（Goal）**：在不破坏 CE 现有"无收款 / 反遥测 / 自托管"原则的前提下，补齐 15 项**高频痛点 + 可观测性 + 差异化**功能，让 x-panel-ce 在日常使用层面显著优于上游 X-Panel。

**架构原则（Architecture）**：
1. **加法、不重构** —— 每项功能新增文件 / 新增字段 / 新增路由，不动他人现有逻辑。
2. **每项独立可 ship** —— 任意一项失败回滚不影响其他项；按 release 串行发布。
3. **零外部网络依赖** —— 不引入 telemetry、不调用第三方 API（除已有的 GitHub Releases / acme.sh / xray geo 数据）。
4. **谨慎 timing / threading / resource**（测试工程师底线）—— 任何 goroutine / cron / 文件锁都要标注，并配最小化的并发安全实现。

**技术栈（Tech Stack）**：Go 1.21+ / Gin / GORM / Vue.js 2 / Ant Design Vue 1.x / Tornado(webssh) / Bash / sqlite3 CLI / xray-core。

---

## 0. 编号约定与依赖图

ROADMAP §2 规则："新增条目从 100 起编号，避免与上游冲突"。本计划共 15 项，编号 **#100-#114**。

| # | 名称 | 优先级 | 依赖前置 | 涉及 timing/threading | Release |
|---|---|---|---|---|---|
| 100 | 自动 DB 备份 + 保留 N 天 | P0 | — | ✓ goroutine + cron | **ce-1.1.0** |
| 101 | 批量导出节点链接 / 订阅 / 二维码 | P0 | — | — | **ce-1.1.0** |
| 102 | 批量启用 / 禁用 / 删除 inbound | P0 | — | ✓ 批量请求节流 | **ce-1.1.0** |
| 103 | 入站健康度可视化（绿/黄/红） | P0 | — | ✓ 节流轮询 | **ce-1.2.0** |
| 104 | TG `/health` 命令 | P0 | Stage 3 已就位 | — | **ce-1.2.0** |
| 105 | 入站标签 / 分组（Tag） | P1 | — | ✓ GORM auto-migrate | **ce-1.3.0** |
| 106 | 内置 Clash / sing-box 订阅转换 | P1 | — | — | **ce-1.3.0** |
| 107 | 节点 RTT / TCP 自检 | P1 | — | ✓ goroutine 并发 dial | **ce-1.3.0** |
| 108 | Reality 公私钥 / clients CSV 导出 | P1 | — | — | **ce-1.3.0** |
| 109 | WARP 出站一键开关 | P1 | — | — | **ce-1.4.0** |
| 110 | 入站克隆到新端口 | P2 | — | — | **ce-1.4.0** |
| 111 | DB 加密导出 / 导入（跨 VPS 迁移） | P2 | #100 | — | **ce-1.4.0** |
| 112 | Cron 任务可视化页签 | P2 | — | — | **ce-1.2.0** |
| 113 | 面板内嵌 webssh iframe | P2 | Stage 2 #7 已就位 | — | **ce-1.4.0** |
| 114 | 入站到期 / 流量预警 TG 通知 | P2 | Stage 3 已就位 | ✓ cron | **ce-1.4.0** |

**依赖关系图**：

```
ce-1.1.0  ┬─ #100 (DB 备份)         ──────┐
          ├─ #101 (批量导出)               │
          └─ #102 (批量操作)               │
                                           │
ce-1.2.0  ┬─ #103 (健康度)                 │
          ├─ #104 (TG /health)             │
          └─ #112 (Cron 可视化)            │
                                           ▼
ce-1.3.0  ┬─ #105 (Tag)                  ┌─ #111 依赖 #100 备份能力
          ├─ #106 (订阅转换)              │
          ├─ #107 (RTT 自检)              │
          └─ #108 (CSV 导出)              │
                                          │
ce-1.4.0  ┬─ #109 (WARP)                  │
          ├─ #110 (克隆)                  │
          ├─ #111 (DB 迁移) ◄─────────────┘
          ├─ #113 (webssh iframe)
          └─ #114 (到期预警)
```

---

## 1. Release 切分与发布节奏

| Release | 主题 | 编号 | 估时 | 验收门禁 |
|---|---|---|---|---|
| **ce-1.1.0** | 稳定性 + 高频痛点 | #100 / #101 / #102 | ~10h（1.5 天） | 升级一次面板不再担心 db 损坏 + 客户端配置发送从 30 次点击降到 1 次 |
| **ce-1.2.0** | 可观测性 + 远程管理 | #103 / #104 / #112 | ~12h（1.5 天） | 管理员在路上能掌握 VPS 状态 + 孤儿 inbound 立即可见 |
| **ce-1.3.0** | 差异化能力 | #105 / #106 / #107 / #108 | ~16h（2 天） | CE 用户体验显著优于上游 + 多客户端格式无缝 |
| **ce-1.4.0** | 高级运维 | #109 / #110 / #111 / #113 / #114 | ~14h（2 天） | 跨 VPS 迁移 + 出口路由 + 客户预警全自动化 |

**总计**：~52 小时 ≈ 7 个工作日。

**发布节奏建议**：每完成一个 release 在生产 VPS 验证 24-48h（用 `tools/ce-vps-smoke.sh`）后再开下一个 release，避免连环 hotfix（参考 ce-1.0.1/1.0.2 教训）。

---

## 2. 功能详情（按编号）

### ce-1.1.0 — 稳定性 + 高频痛点

---

#### #100 自动 DB 备份 + 保留 N 天

**痛点**：用户最怕"升级把 `/etc/x-ui/x-ui.db` 弄坏"。ce-1.0.1 toml 启动崩溃 + ce-1.0.2 user_id bug 都已经踩过，下一次出问题不一定有时间手动救数据库。

**文件**：
- 创建：`web/service/db_backup.go`
- 修改：`main.go`（Start 钩子注册 cron + 启动时立即备份一次）
- 测试：`web/service/db_backup_test.go`

**关键设计**：
- 用 `sqlite3 .backup` 命令而**不是** `cp`（避免文件锁竞争损坏文件）
- 备份路径：`/etc/x-ui/backup/x-ui-YYYYMMDD-HHMMSS.db.gz`
- 保留策略：最近 7 份 + 每周一份保留 4 周
- cron：每天 03:00 一次
- 启动时：异步 `go backupDBOnce()`（不阻塞 web server 启动）

**threading 注意**：单例 mutex 保护，避免 cron 触发 + 启动钩子并发同名文件名冲突。

**估时**：2h
**验收命令**：
```bash
# 1. 启动服务后 5 秒内应有备份生成
sudo systemctl restart x-ui && sleep 5
ls -lh /etc/x-ui/backup/

# 2. 备份文件可解压且 sqlite header 完整
zcat /etc/x-ui/backup/x-ui-*.db.gz | head -c 16 | xxd
# 期望: 00000000: 5351 4c69 7465 2066 6f72 6d61 7420 3300  SQLite format 3.

# 3. 30 天后保留策略生效（人工写测试假数据）
ls /etc/x-ui/backup/ | wc -l   # ≤ 11
```

---

#### #101 批量导出节点链接 / 订阅 / 二维码

**痛点**：30 条 reality 一键建完，要逐条点开 → 二维码 → 复制 → 关闭 × 30 次。

**文件**：
- 修改：`web/html/inbounds.html`（列表头加"批量导出"按钮 + 模态）
- 修改：`web/controller/inbound.go`（加 `POST /panel/inbound/batchExport` 路由）
- 修改：`web/service/inbound.go`（加 `BatchExportLinks(ids []int) ([]string, error)`，复用现有 `genLink` 逻辑）

**关键设计**：
- **纯查询**，无副作用 → 零数据风险
- 模态展示三个 tab：
  - vless 链接列表（一行一条，"复制全部"按钮）
  - 合并的订阅 base64（v2rayN 直接订阅）
  - 二维码网格（4×N 布局，扫码用）

**估时**：3-4h
**验收命令**：
```bash
# 1. API 返回所有勾选 inbound 的 vless 链接
curl -X POST -b cookie.txt http://127.0.0.1:54321/panel/inbound/batchExport \
  -d '{"ids":[1,2,3]}' -H 'Content-Type: application/json'

# 2. 复制全部按钮把链接放进剪贴板（前端 navigator.clipboard）
# 在 v2rayN 粘贴 → 应识别 N 个 vless 节点
```

---

#### #102 批量启用 / 禁用 / 删除 inbound

**痛点**：30 条 reality 测试完不想要了，得点 30 次"删除→确认"。

**文件**：
- 修改：`web/html/inbounds.html`（列表加复选框列 + 底部批量操作栏）
- 复用：现有 `web/controller/inbound.go` 的 `delInbound` / `updateInbound` 路由（**不**新增后端 API，前端 `Promise.allSettled` 串发）

**关键设计**：
- 串发节流：每 50ms 一个请求，避免 SQLite 写锁竞争（**timing 注意**）
- 失败聚合：N 个删除请求 M 个失败 → 一次性提示"成功 N-M / 失败 M（点查看详情）"
- **不**给批量操作套外层事务（`AddInbound` 内部已有事务，外层套会嵌套死锁 —— 这是教训）

**估时**：2-3h
**验收命令**：
```bash
# 1. 创建 30 条 reality 后批量删除，全部消失
sqlite3 /etc/x-ui/x-ui.db "SELECT count(*) FROM inbounds"
# 期望: 0

# 2. 中途有一条删除失败（手动锁 db 模拟），UI 应正确显示部分成功
```

---

### ce-1.2.0 — 可观测性 + 远程管理

---

#### #103 入站健康度可视化（绿/黄/红圆点）

**痛点**：你刚踩过 30 条 inbound 落库但前端不显示的 user_id bug。如果列表行能展示"xray 是否真的认这个 inbound"，第一时间就能发现孤儿数据。

**文件**：
- 创建：`web/service/inbound_health.go`
- 修改：`web/html/inbounds.html`（每行加一个 status 圆点）
- 修改：`web/controller/inbound.go`（加 `GET /panel/inbound/health` 路由）

**关键设计**：
- **三态**：
  - 🟢 绿：DB 有 + xray config 包含 + 端口 LISTEN
  - 🟡 黄：DB 有 + xray 包含但端口未 LISTEN（可能配置错误）
  - 🔴 红：DB 有但 xray 不认（**孤儿数据**，user_id bug 复现立即可见）
- **节流**：30s 一次或手动刷新触发，**不要每秒轮询**（VPS CPU 友好）
- 实现：用 `net.Listen("tcp", "127.0.0.1:N")` 探测端口被占用？还是 `ss -tlnp`？→ 选 `ss`（更轻、零侧效）

**threading 注意**：goroutine 池子大小限制（最多并发 10 个端口探测），用 `errgroup` 控制。

**估时**：4-6h
**验收命令**：
```bash
# 1. 创建一条 user_id=0 的孤儿数据（模拟 bug 复现）
sqlite3 /etc/x-ui/x-ui.db "INSERT INTO inbounds(user_id, port, protocol, enable) VALUES(0, 30099, 'vless', 1)"

# 2. 前端列表应显示红色圆点（用户登录后看不到该行 → 但 admin 可以通过 health 接口看到）
curl -b cookie.txt http://127.0.0.1:54321/panel/inbound/health
# 期望: [{"id":N,"port":30099,"status":"red","reason":"orphan_user_id"}]
```

---

#### #104 TG `/health` 命令

**痛点**：管理员在路上想知道 VPS 状态得 SSH 登。

**文件**：
- 修改：`web/service/tgbot.go`（新增 case "health"）
- 复用：现有 `web/service/server.go:GetStatus`

**关键设计**：
- 发送一条 markdown 消息：
  - CPU / MEM / Disk
  - xray uptime + 进程内存
  - 最近 24h 流量 top 5 inbounds
  - 当前活跃连接数（`ss -ant | grep ESTAB | wc -l`）
- 严格本地：不联网查任何外部 API

**估时**：2-3h
**验收命令**：
```bash
# 在 TG 给 bot 发 /health
# 期望返回完整 markdown 报告，3 秒内回复
```

---

#### #112 Cron 任务可视化页签

**痛点**：流量重置 / TG 报告 / DB 备份这几个 cron 现在都是隐式的。用户排查"为什么没自动重置"得读源码。

**文件**：
- 创建：`web/html/cron_jobs.html`
- 修改：`web/service/cron_inspector.go`（新建，从 `robfig/cron` Entry 列表里拉信息）
- 修改：`web/controller/index.go`（注册路由）
- 修改：`web/html/navigation.html`（侧边栏加入口）

**关键设计**：
- 表格列：任务名 / cron 表达式 / 上次执行时间 / 下次执行时间 / 上次执行结果（成功/失败/原因）
- 只读，**不允许**前端修改 cron（避免误操作打挂面板）

**估时**：4h
**验收命令**：
```bash
# 在 /cron_jobs 页应能看到 #100 DB 备份任务，下次执行时间是明天 03:00
```

---

### ce-1.3.0 — 差异化能力

---

#### #105 入站标签 / 分组（Tag）

**痛点**：30 条 reality 在列表里平铺没法分类（哪些给客户 A、哪些给客户 B）。

**文件**：
- 修改：`database/model/model.go`（`Inbound` 结构体加 `GroupTag string` 字段）
- 修改：`database/db.go`（auto-migrate 在已有 `db.AutoMigrate` 调用里自动添加列）
- 修改：`web/html/inbounds.html`（列表头加 tag 筛选 + tag 列）
- 修改：`web/html/modals/inbound_modal.html`（编辑表单加 tag 输入框，autocomplete 已存在 tag 列表）

**关键设计**：
- **DB schema 变更** —— 必须走 GORM auto-migrate，不要写手动 SQL
- 默认空字符串 → 视为 "ungrouped" 组（兼容老数据）
- 前端用 `localStorage` 记住"上次选择的 tag 筛选条件"

**风险**：sqlite ALTER TABLE ADD COLUMN 是原子的、零风险，但**必须备份 db**（已由 #100 兜底）。

**估时**：4-5h
**验收命令**：
```bash
# 1. 升级后老数据 group_tag 全为空字符串
sqlite3 /etc/x-ui/x-ui.db "SELECT id, group_tag FROM inbounds LIMIT 5"

# 2. 前端给 30 条 inbound 打 tag={"customer-a","customer-b"}，列表筛选生效
```

---

#### #106 内置 Clash / sing-box 订阅转换

**痛点**：你已经移除上游 sublink/subconverter（依赖闭源），CE 用户的客户端格式没法自动转。

**文件**：
- 创建：`web/service/sub_converter.go`（纯 Go，**不引入** subconverter 这种重型依赖）
- 修改：`web/controller/sub.go`（加 `/sub/clash/<token>` 和 `/sub/singbox/<token>` 路由）
- 测试：`web/service/sub_converter_test.go`（用上游 v2rayN 项目里的样例 vless URL 做单元测试）

**关键设计**：
- **仅支持** vless / vmess → clash yaml + sing-box json（覆盖 CE 90% 用户场景）
- 不支持 ssr/trojan/hysteria（如果以后用户要再加；YAGNI）
- 全本地：纯字符串拼接 + 标准库 `encoding/json` + `gopkg.in/yaml.v3`（已在 go.mod）
- 路径与 token：复用现有 sub 系统的鉴权（`web/service/sub_service.go`）

**估时**：1-2 天
**验收命令**：
```bash
# 1. 拉一份 clash 订阅，喂给 mihomo 解析应成功
curl http://127.0.0.1:2096/sub/clash/<token> > /tmp/clash.yaml
mihomo -t -d /tmp -f /tmp/clash.yaml
# 期望: configuration file test is successful

# 2. 同样测 sing-box
curl http://127.0.0.1:2096/sub/singbox/<token> > /tmp/singbox.json
sing-box check -c /tmp/singbox.json
# 期望: 无错误输出
```

---

#### #107 节点 RTT / TCP 自检

**痛点**：30 条 reality 部署完，不知道哪条端口走得通畅。

**文件**：
- 创建：`web/service/inbound_rtt.go`
- 修改：`web/html/inbounds.html`（列表头加"RTT 自检"按钮）
- 修改：`web/controller/inbound.go`（加 `POST /panel/inbound/rttCheck` 路由）

**关键设计**：
- **纯本机** `net.DialTimeout("tcp", "127.0.0.1:port", 1*time.Second)`
- 绝不引入 ping / mtr / traceroute 等系统命令（CE 不依赖外部工具）
- 前端展示：表格列出每条 inbound 的"建连耗时（ms）"+ 颜色（< 5ms 绿、5-50ms 黄、> 50ms 红）

**threading 注意**：`errgroup` 限制并发 10，避免 30 条同时 dial 把 ulimit 打爆。

**估时**：2h
**验收命令**：
```bash
# 1. 自检按钮触发，3 秒内返回所有 30 条端口的 RTT
# 2. 关掉 xray 后，RTT 应全部 timeout（红色）
sudo systemctl stop xray
# 前端再次自检 → 全部红
sudo systemctl start xray
```

---

#### #108 Reality 公私钥 / clients CSV 导出

**痛点**：给客户运维交付节点时格式不统一，用户得自己用 Excel 整理。

**文件**：
- 修改：`web/html/inbounds.html`（列表头加"导出 CSV"按钮）
- 修改：`web/controller/inbound.go`（加 `POST /panel/inbound/exportCSV` 路由，`Content-Type: text/csv`）
- 修改：`web/service/inbound.go`（加 `ExportInboundsCSV(ids []int) (string, error)`）

**关键设计**：
- CSV 列：`id, port, remark, protocol, sni, pubkey, shortId, clientId, email, group_tag`
- 字符编码 UTF-8 with BOM（兼容 Excel 中文显示）
- 鉴权：admin 才能导出（contains 私钥的话），普通用户用 #101 批量导出链接

**估时**：1-2h
**验收命令**：
```bash
curl -X POST -b cookie.txt http://127.0.0.1:54321/panel/inbound/exportCSV \
  -d '{"ids":[1,2,3]}' -H 'Content-Type: application/json' \
  -o /tmp/inbounds.csv
# 期望: CSV 头 BOM + 10 列，行数 = 选择的 inbound 数 + 1（表头）
```

---

### ce-1.4.0 — 高级运维

---

#### #109 WARP 出站一键开关

**痛点**：想让 VPS 出口走 cloudflare WARP（隐藏真实 IP / 解锁流媒体）现在得手改 xray.json。

**文件**：
- 修改：`x-ui.sh`（show_menu 新增 case 30: `warp_install_outbound`）
- 创建：`x-ui.sh:warp_install_outbound`（参考 Stage 2 #29 内核调优脚本风格：dry-run + 备份 + 回滚）

**关键设计**：
- 装 wgcf（已有 release binary）→ 生成 wireguard outbound → 注入 xray 模板 → 重启 xray
- **回滚**：备份 `/usr/local/x-ui/bin/config.json` 到 `/usr/local/x-ui/bin/config.json.bak.<ts>`
- 失败检测：xray 启动后健康检查 30s 内未通过 → 自动回滚

**风险**：脚本要严格 dry-run preview 一次，让用户确认（**不能默认 Y**）。

**估时**：3-4h
**验收命令**：
```bash
# 1. 启用 WARP 后，curl ifconfig.me 应返回 cloudflare IP（与 VPS 真实 IP 不同）
# 2. 关闭 WARP 后，curl ifconfig.me 返回 VPS 真实 IP
# 3. 中途 xray 启动失败 → 自动回滚，恢复原 config.json
```

---

#### #110 入站克隆到新端口

**痛点**：想新建一条结构相同的 inbound 得重新填表。

**文件**：
- 修改：`web/html/inbounds.html`（每行加"克隆"按钮）
- 修改：`web/controller/inbound.go`（加 `POST /panel/inbound/clone`）
- 修改：`web/service/inbound.go`（加 `CloneInbound(id int, newPort int) (*model.Inbound, error)`）

**关键设计**：
- 端口冲突自动加 1 直到找到空闲端口（最多尝试 100 次，避免死循环）
- 克隆复制：除了 `id`、`port`、`up`、`down` 之外的所有字段
- **重新生成** Reality 的 keypair / shortId / clientId（避免 UUID 复用）

**估时**：1h
**验收命令**：
```bash
# 1. 克隆 inbound 1 到新端口 → 应有新行，UUID/pubkey 不同
# 2. 端口被占 → 自动选下一个空闲端口
```

---

#### #111 DB 加密导出 / 导入（跨 VPS 迁移）

**痛点**：换 VPS 时迁移 db 得手动 scp + 改文件名。

**文件**：
- 创建：`x-ui.sh:db_export` / `x-ui.sh:db_import` 两个新菜单项
- 修改：`x-ui.sh:show_menu`（case 31 / 32）
- 依赖：#100（备份能力）

**关键设计**：
- **导出**：
  ```
  tar czf x-ui-<hostname>-<ts>.tar.gz \
    /etc/x-ui/x-ui.db \
    /usr/local/x-ui/bin/config.json \
    /etc/x-ui/access.log
  + sha256sum > x-ui-<hostname>-<ts>.sha256
  + 提示用户用 GPG / 7z 自己加密（不在脚本里强制），避免 CE 引入加密依赖
  ```
- **导入**：
  - 校验 sha256
  - **强制要求** 备份当前 db 到 `#100` 备份目录
  - 停 x-ui → 复制 → 启 x-ui → 健康检查 30s

**风险**：导入路径如果错了会**覆盖现有 db**，必须有 `--dry-run` 模式 + 强确认提示。

**估时**：3h
**验收命令**：
```bash
# 1. 在 VPS-A 导出
sudo /usr/bin/x-ui   # 选 31 → 导出 → 得到 /tmp/x-ui-vpsA-20260427.tar.gz

# 2. scp 到 VPS-B
scp /tmp/x-ui-*.tar.gz vpsB:/tmp/

# 3. 在 VPS-B 导入
sudo /usr/bin/x-ui   # 选 32 → 导入 → 健康检查通过

# 4. VPS-B 应能访问到 VPS-A 的所有 inbound 数据
sqlite3 /etc/x-ui/x-ui.db "SELECT count(*) FROM inbounds"
```

---

#### #113 面板内嵌 webssh iframe

**痛点**：webssh 已经 systemd 化（Stage 2 #7 done），但只暴露 127.0.0.1。每次还得 SSH tunnel 才能用。

**文件**：
- 创建：`web/html/webssh.html`（iframe 嵌入 127.0.0.1:webssh-port，通过面板反代）
- 修改：`web/controller/index.go`（加 `/webssh` 路由 + admin 鉴权 + 反代 127.0.0.1:8889）
- 修改：`web/html/navigation.html`（侧边栏加入口，仅 admin 可见）

**关键设计**：
- 面板做反代（用 `httputil.ReverseProxy`），把 `/webssh/*` 转发到 `127.0.0.1:8889/*`
- 严格 admin 才能访问（已有的 session middleware）
- iframe sandbox 属性：`allow-scripts allow-same-origin`（webssh 需要这两个）

**风险**：反代路径要对 WebSocket upgrade 透传（webssh 用 ws）。

**估时**：2h
**验收命令**：
```bash
# 1. 访问 https://panel.example.com/webssh → iframe 加载 webssh 终端
# 2. 在终端执行 ls /etc/x-ui → 应有响应
# 3. 普通 user 访问 /webssh → 401
```

---

#### #114 入站到期 / 流量预警 TG 通知

**痛点**：客户的入站快到期了 / 流量快用完了，没人提醒，等用户投诉才发现。

**文件**：
- 修改：`web/service/tgbot.go`（新增 cron job: `checkInboundWarning`，每天 09:00 执行）
- 复用：现有 `Inbound.ExpiryTime` 和 `Total` / `Up+Down` 字段

**关键设计**：
- 触发条件：
  - `ExpiryTime - now < 24h` → 发"24 小时内到期"
  - `(Up+Down) / Total > 0.9` → 发"流量已用 90%"
- 防止刷屏：每条 inbound 每个预警类型 24h 内只发一次（用 setting 表记录最后通知时间）
- **timing 注意**：cron 一次性扫所有 inbound，要分批 N=50 处理避免阻塞 TG bot 主循环

**估时**：2h
**验收命令**：
```bash
# 1. 手动改 db: UPDATE inbounds SET expiry_time = (strftime('%s','now')+3600)*1000 WHERE id=1
# 2. 等下次 09:00 cron → TG 收到"inbound 1 即将在 1 小时后到期"消息
```

---

## 3. 验收门禁（每个 Release 都要做）

每个 release 推 tag 之前，必须在生产 VPS 上跑：

```bash
cd /home/jack/work/x-panel-ce
bash tools/ce-vps-smoke.sh    # 现有冒烟脚本
```

如果是 `ce-1.x.0` 大版本，**额外**做：

1. **升级路径验证** —— 从 `ce-1.(x-1).0` 升级到 `ce-1.x.0`，db 自动 migrate 不爆炸
2. **回滚路径验证** —— 从 `ce-1.x.0` 回滚到 `ce-1.(x-1).0`，老 db schema 还能用（兼容）
3. **24h 稳定性观察** —— 不重启服务，观察 cron / goroutine 是否泄漏（`top` 看 RSS）

---

## 4. 风险与防范

| 风险类型 | 出现项 | 防范措施 |
|---|---|---|
| **DB schema 变更** | #105 (Tag) | GORM auto-migrate + #100 备份兜底 |
| **goroutine 泄漏** | #100 / #103 / #107 / #114 | 严格 `defer cancel()` + errgroup 限制 + 单测覆盖 |
| **文件锁竞争** | #100 (sqlite backup) | 用 `sqlite3 .backup` 命令而非 `cp` |
| **批量请求并发** | #102 / #107 | 节流 50ms/请求 + 并发上限 10 |
| **WebSocket 反代** | #113 | `httputil.ReverseProxy` 默认支持 + 验证 webssh 实际可用 |
| **xray 重启回滚** | #109 | dry-run + 备份 + 健康检查 30s |
| **DB 导入覆盖** | #111 | 强制提前备份 + 强确认提示 + dry-run 模式 |

---

## 5. 不做事项（明确边界）

下列功能在 Stage 7 内**明确不做**，避免破坏 CE 原则：

| 不做 | 原因 |
|---|---|
| 任何"上报到中心服务器/统计接口" | 违反 Stage 6 永久反遥测承诺 |
| "自动巡检 + 自动换 IP" | 跟 VPS 厂商 API 强耦合，破坏自托管原则 |
| "积分 / 签到 / 抽奖" | ROADMAP §3.6 已明确不实现（#26/#27/#28）|
| "在线支付 / 购买机器人" | ROADMAP §3.6 已明确不实现（#32）|
| "aff 推广链接 / 广告位" | ce-1.0.3 刚刚清完，不能开倒车 |
| 重写 xray 配置生成层 | 会破坏上游兼容，user 升级 db 会爆 |
| 引入 Redis / MySQL 等外部存储 | CE 坚持 sqlite 单文件、零运维原则 |
| 重写订阅系统的鉴权 | #106 复用现有 sub_service.go token 即可 |

---

## 6. 执行交接（Execution Handoff）

本规划已落盘。两种执行选项：

**1. 子代理逐项推进（subagent-driven，本会话内）**
- 我每完成一项即让你 review，节奏快、迭代密
- 适合 ce-1.1.0（3 项 P0 是高频高反馈）

**2. 独立会话批量执行（executing-plans，新会话）**
- 你开新会话，引用本计划文件，我按 release 整批推
- 适合 ce-1.3.0 / ce-1.4.0 那种复杂功能（避免单会话过载）

**建议混合**：
- ce-1.1.0 + ce-1.2.0 用方案 1（你近期参与度高，能立即 review）
- ce-1.3.0 + ce-1.4.0 用方案 2（差异化功能 + 高级运维，单项工作量大）

---

**最后更新**：2026-04-26
**计划负责人**：维护者（hehelove）
**预计完成时间**：自 ce-1.0.3 之后 7 个工作日
