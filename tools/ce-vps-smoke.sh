#!/usr/bin/env bash
# ============================================================
# x-panel-ce VPS Smoke Test（部署后自检脚本）
# ------------------------------------------------------------
# 用途：在 VPS 上跑一次，验证 x-panel-ce 部署对应 ROADMAP 里
#       Stage 0/2/3/4 的关键交付，并扫描隐私后门 / 抽奖框架 / 上游商业域名
#       等"绝不应残留"的指纹。脚本幂等、只读（除写日志到 /tmp/）。
#
# 用法：
#   bash ce-vps-smoke.sh                # 二进制 + runtime 全套检查
#   bash ce-vps-smoke.sh --no-runtime   # 只验证已编译二进制，不检查端口/服务
#   bash ce-vps-smoke.sh --bin /path    # 指定 x-ui 二进制位置
#   bash ce-vps-smoke.sh --port 2053    # 指定面板端口（避免读 setting）
#
# 期望输出：报告写到 /tmp/x-panel-ce-smoke-<时间戳>.log；
#          stdout 同步打印；
#          PASS=绿色 / WARN=黄色 / FAIL=红色；
#          出现 FAIL 退出码 1，否则 0。
#
# 详见 docs/VPS-VERIFICATION-CHECKLIST.md。
# ============================================================

# 注意：不用 set -e（让所有检查跑完，最终汇总）；用 -uo 防止变量未定义
set -uo pipefail

# ------- 默认值 / 参数解析 -------
BIN_PATH="${BIN_PATH:-/usr/local/x-ui/x-ui}"
DB_PATH="${DB_PATH:-/etc/x-ui/x-ui.db}"
CHECK_RUNTIME=1
PANEL_PORT=""
TG_SAMPLE_SEC="${TG_SAMPLE_SEC:-30}"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --no-runtime) CHECK_RUNTIME=0; shift ;;
        --bin)        BIN_PATH="$2"; shift 2 ;;
        --db)         DB_PATH="$2"; shift 2 ;;
        --port)       PANEL_PORT="$2"; shift 2 ;;
        --tg-sec)     TG_SAMPLE_SEC="$2"; shift 2 ;;
        -h|--help)
            sed -n '2,20p' "$0"
            exit 0
            ;;
        *)
            echo "未知参数: $1（用 --help 看用法）" >&2
            exit 2
            ;;
    esac
done

REPORT="/tmp/x-panel-ce-smoke-$(date +%Y%m%d-%H%M%S).log"

# ------- 颜色 -------
if [[ -t 1 ]]; then
    RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[0;33m'; BLUE='\033[0;34m'; PLAIN='\033[0m'
else
    RED=''; GREEN=''; YELLOW=''; BLUE=''; PLAIN=''
fi

# ------- 计数 + 打印工具 -------
TOTAL=0; PASSED=0; FAILED=0; WARNED=0
pass()    { TOTAL=$((TOTAL+1)); PASSED=$((PASSED+1)); echo -e "  [${GREEN}PASS${PLAIN}] $1" | tee -a "$REPORT"; }
fail()    { TOTAL=$((TOTAL+1)); FAILED=$((FAILED+1)); echo -e "  [${RED}FAIL${PLAIN}] $1" | tee -a "$REPORT"; }
warn()    { TOTAL=$((TOTAL+1)); WARNED=$((WARNED+1)); echo -e "  [${YELLOW}WARN${PLAIN}] $1" | tee -a "$REPORT"; }
info()    { echo -e "  [INFO] $1" | tee -a "$REPORT"; }
section() { echo "" | tee -a "$REPORT"; echo -e "${BLUE}===== $1 =====${PLAIN}" | tee -a "$REPORT"; }

# ------- 报告头 -------
{
    echo "========================================"
    echo "x-panel-ce VPS Smoke Test 报告"
    echo "========================================"
    echo "时间(UTC): $(date -u +%FT%TZ)"
    echo "时间(本地): $(date +'%F %T %Z')"
    echo "主机: $(hostname)"
    echo "执行用户: $(id -un) (uid=$(id -u))"
    echo "二进制路径: $BIN_PATH"
    echo "数据库路径: $DB_PATH"
    echo "运行时检查: $([ "$CHECK_RUNTIME" = 1 ] && echo '启用' || echo '禁用')"
    echo "----------------------------------------"
} | tee "$REPORT"

# ============================================================
# Section 1: 环境探测
# ============================================================
section "1. 环境探测"

if [[ -f /etc/os-release ]]; then
    # shellcheck source=/dev/null
    . /etc/os-release
    info "OS: ${PRETTY_NAME:-$ID $VERSION_ID}"
else
    warn "无 /etc/os-release（非主流 Linux？）"
fi

info "内核: $(uname -r)"
info "架构: $(uname -m)"
if command -v nproc >/dev/null; then
    info "CPU: $(nproc) cores"
fi
if [[ -r /proc/meminfo ]]; then
    info "内存: $(awk '/MemTotal/ {printf "%.1f GB", $2/1024/1024}' /proc/meminfo)"
fi
info "磁盘根分区: $(df -h / 2>/dev/null | awk 'NR==2 {print $4 " 可用 / " $2 " 总"}')"

# 时区探测：先尝试 readlink /etc/localtime（最稳健），再 fallback 到 date / cat
TZ_NAME=""
if [[ -L /etc/localtime ]]; then
    TZ_NAME=$(readlink /etc/localtime 2>/dev/null | sed 's|.*zoneinfo/||')
fi
[[ -z "$TZ_NAME" && -f /etc/timezone ]] && TZ_NAME=$(cat /etc/timezone 2>/dev/null)
[[ -z "$TZ_NAME" ]] && TZ_NAME=$(date +%Z 2>/dev/null)
info "时区: ${TZ_NAME:-unknown}"

# ROADMAP #29 内核调优（BBR 验证）
if [[ -r /proc/sys/net/ipv4/tcp_congestion_control ]]; then
    CC=$(cat /proc/sys/net/ipv4/tcp_congestion_control)
    if [[ "$CC" == "bbr" ]]; then
        pass "TCP 拥塞控制 = bbr（ROADMAP #29 内核调优已生效）"
    else
        warn "TCP 拥塞控制 = $CC（如未在 x-ui 菜单跑过 BBR 调优，这是正常的）"
    fi
fi

# ============================================================
# Section 2: 二进制存在性 + 静态健康
# ============================================================
section "2. 二进制健康"

if [[ ! -e "$BIN_PATH" ]]; then
    fail "x-ui 二进制不存在: $BIN_PATH"
    info "  -> 先跑 install.sh 或源码 go build"
    if [[ $CHECK_RUNTIME -eq 1 ]]; then
        warn "无二进制 → 跳过后续 runtime 部分"
        CHECK_RUNTIME=0
    fi
else
    pass "x-ui 二进制存在: $BIN_PATH"

    SIZE=$(stat -c%s "$BIN_PATH" 2>/dev/null || ls -l "$BIN_PATH" | awk '{print $5}')
    info "大小: $((SIZE / 1024 / 1024)) MB"

    # 静态链接检测：CE release artifact 应该是静态二进制
    if file "$BIN_PATH" 2>/dev/null | grep -q 'statically linked'; then
        pass "静态链接（不依赖宿主机 glibc）"
    elif ldd "$BIN_PATH" 2>&1 | grep -qE 'not a dynamic|statically linked'; then
        pass "静态二进制（ldd 报告 not dynamic）"
    else
        warn "动态链接（如果是源码 go build 的，正常；release artifact 应是静态）"
    fi

    # 版本字串
    if VER_OUT=$("$BIN_PATH" -v 2>&1); then
        info "版本: $VER_OUT"
    fi
fi

# ============================================================
# Section 3: 反检测扫描（CE 路线图核心交付验证）
# ============================================================
section "3. 反检测：二进制字串扫描"

if [[ ! -e "$BIN_PATH" ]]; then
    info "跳过：二进制不存在"
elif ! command -v strings >/dev/null 2>&1; then
    warn "未安装 binutils/strings —— 跳过二进制字串扫描"
elif true; then

    # 3.1 抽奖框架（commit 8c8d702e 整段移除）
    if strings "$BIN_PATH" | grep -iE 'runlottery|sendlotterygame|lottery_play|lottery_skip|LOTTERY_STICKER' > /dev/null 2>&1; then
        fail "二进制中残留 lottery 框架字串（CE 应已清理；提示重新 build）"
        strings "$BIN_PATH" | grep -iE 'runlottery|sendlotterygame|lottery_play' | head -3 | sed 's/^/      /' | tee -a "$REPORT"
    else
        pass "未发现 lottery 框架字串（commit 8c8d702e 验证 OK）"
    fi

    # 3.2 上游商业域名
    if strings "$BIN_PATH" | grep -iE 'xeefei\.com|xui\.cool' > /dev/null 2>&1; then
        fail "二进制中发现上游商业域名（xeefei.com / xui.cool）"
    else
        pass "未发现上游商业域名"
    fi

    # 3.3 隐私后门标记（Stage 0.1 commit 9c5599d2 已清理；这里只是断言不复现）
    # 注意：上游硬编码的 chat ID 和 bot token 字面值已删除，但 grep 数字 ID 容易误伤；
    # 这里改用 CE 路线图明确禁用的 marker 字串
    if strings "$BIN_PATH" | grep -iE 'CENTRAL_(REPORT|MONITOR)|UPSTREAM_REPORT_CHAT|REPORT_TO_DEVELOPER' > /dev/null 2>&1; then
        fail "二进制中发现中央上报 marker（隐私后门可能未清理）"
    else
        pass "未发现中央上报 marker"
    fi

    # 3.4 上游脚本里也搜一下源码：CE 路线图不应保留中央上报的 channel id 长数字
    # 这条仅在文件系统中查（不在二进制里），跳到 Section 4 与文件系统一起检查
    :
fi

# ============================================================
# Section 4: 文件系统残留扫描
# ============================================================
section "4. 文件系统残留"

# install.sh / x-ui.sh 等 .sh 脚本里检查
FOUND_ANY=0
for FILE in /usr/local/x-ui/x-ui.sh /usr/bin/x-ui /usr/local/x-ui/install.sh; do
    if [[ -f "$FILE" ]]; then
        FOUND_ANY=1
        if grep -iE '抽奖|lottery_play|runLotteryDraw' "$FILE" > /dev/null 2>&1; then
            fail "$FILE 残留 lottery 字串"
        else
            pass "$FILE 干净"
        fi
    fi
done
[[ $FOUND_ANY -eq 0 ]] && info "跳过：未发现 /usr/local/x-ui/ 安装目录"

# ============================================================
# Section 5: systemd 服务（runtime）
# ============================================================
if [[ $CHECK_RUNTIME -eq 1 ]]; then
    section "5. Runtime: systemd 服务状态"

    if ! command -v systemctl >/dev/null 2>&1; then
        warn "未安装 systemd（容器/最小化系统？）"
    elif ! systemctl list-unit-files 2>/dev/null | grep -q '^x-ui\.service'; then
        warn "x-ui.service 未注册到 systemd"
    else
        pass "x-ui.service 已注册"

        STATUS=$(systemctl is-active x-ui 2>/dev/null || true)
        if [[ "$STATUS" == "active" ]]; then
            pass "x-ui 服务运行中（active）"
        else
            warn "x-ui 服务状态: ${STATUS:-unknown}"
        fi

        ENABLED=$(systemctl is-enabled x-ui 2>/dev/null || true)
        if [[ "$ENABLED" == "enabled" ]]; then
            pass "x-ui 开机自启已启用"
        else
            warn "x-ui 开机自启未启用 (${ENABLED:-unknown})"
        fi

        # Journalctl 最近 5 分钟错误
        if command -v journalctl >/dev/null 2>&1; then
            ERR_COUNT=$(journalctl -u x-ui --since '5 minutes ago' --no-pager 2>/dev/null | grep -ciE 'panic|fatal|error' || true)
            if [[ "${ERR_COUNT:-0}" -eq 0 ]]; then
                pass "近 5 分钟无 panic/fatal/error 日志"
            else
                warn "近 5 分钟日志含 $ERR_COUNT 处 panic/fatal/error（journalctl -u x-ui 查看）"
            fi
        fi
    fi
fi

# ============================================================
# Section 6: 端口监听
# ============================================================
if [[ $CHECK_RUNTIME -eq 1 ]]; then
    section "6. Runtime: 端口监听"

    # 解析端口
    if [[ -z "$PANEL_PORT" ]] && [[ -e "$BIN_PATH" ]]; then
        # x-ui setting -show 需要能读 db；可能要 root
        if [[ -r "$DB_PATH" ]]; then
            PANEL_PORT=$("$BIN_PATH" setting -show true 2>/dev/null | grep -oE 'port[（(].*[）)]: *[0-9]+' | grep -oE '[0-9]+' | head -1 || true)
        fi
    fi
    if [[ -z "$PANEL_PORT" ]]; then
        PANEL_PORT=2053
        info "未读到 panel 端口配置，按默认 $PANEL_PORT 检查（如不准用 --port 指定）"
    else
        info "面板端口: $PANEL_PORT"
    fi

    if command -v ss >/dev/null 2>&1; then
        if ss -tlnp 2>/dev/null | grep -qE ":${PANEL_PORT}\b"; then
            pass "面板端口 $PANEL_PORT 监听中"
        else
            fail "面板端口 $PANEL_PORT 未监听"
        fi
    elif command -v netstat >/dev/null 2>&1; then
        if netstat -tlnp 2>/dev/null | grep -qE ":${PANEL_PORT}\b"; then
            pass "面板端口 $PANEL_PORT 监听中（netstat）"
        else
            fail "面板端口 $PANEL_PORT 未监听（netstat）"
        fi
    else
        warn "ss 和 netstat 都未安装，跳过端口检查"
    fi
fi

# ============================================================
# Section 7: 数据库 schema
# ============================================================
if [[ $CHECK_RUNTIME -eq 1 ]]; then
    section "7. Runtime: 数据库 schema"

    if [[ ! -f "$DB_PATH" ]]; then
        warn "数据库文件不存在 ($DB_PATH) —— 面板还没初始化或者路径不对"
    elif ! command -v sqlite3 >/dev/null 2>&1; then
        warn "sqlite3 未安装，跳过 schema 验证"
    else
        TABLES=$(sqlite3 "$DB_PATH" '.tables' 2>/dev/null | tr '\n' ' ' || true)

        # CE 期待存在的核心表
        for T in users inbounds settings; do
            if echo "$TABLES" | grep -qw "$T"; then
                pass "表存在: $T"
            else
                fail "表缺失: $T"
            fi
        done

        # CE 期待 NOT 存在的表
        if echo "$TABLES" | grep -qw 'lottery_wins'; then
            warn "残留 lottery_wins 表（旧库迁过来的，可手动 DROP TABLE lottery_wins;）"
        else
            pass "lottery_wins 表已不存在（CE 不再 AutoMigrate）"
        fi

        # 数量摘要
        if echo "$TABLES" | grep -qw 'inbounds'; then
            N_IN=$(sqlite3 "$DB_PATH" 'SELECT COUNT(*) FROM inbounds' 2>/dev/null || echo '?')
            info "inbound 数量: $N_IN"
        fi
        if echo "$TABLES" | grep -qw 'users'; then
            N_USER=$(sqlite3 "$DB_PATH" 'SELECT COUNT(*) FROM users' 2>/dev/null || echo '?')
            info "面板用户数: $N_USER"
        fi
    fi
fi

# ============================================================
# Section 8: HTTP 探测
# ============================================================
if [[ $CHECK_RUNTIME -eq 1 ]] && command -v curl >/dev/null 2>&1; then
    section "8. Runtime: HTTP 接口探测"

    URL="http://127.0.0.1:${PANEL_PORT}/"
    HTTP_CODE=$(curl -ksSI --max-time 5 -o /dev/null -w '%{http_code}' "$URL" 2>/dev/null || echo '000')

    case "$HTTP_CODE" in
        200|301|302|307|308|404)
            pass "HTTP 探测 $URL 响应 $HTTP_CODE（panel 在线）"
            ;;
        000)
            fail "HTTP 探测 $URL 失败（连接拒绝/服务未启动/防火墙）"
            ;;
        *)
            warn "HTTP 探测 $URL 意外状态码: $HTTP_CODE"
            ;;
    esac

    # 抽奖反检测：面板首页 / index 不应返回包含 "抽奖" / "lottery" 的内容
    if [[ "$HTTP_CODE" =~ ^(200|301|302|404)$ ]]; then
        BODY=$(curl -ksSL --max-time 5 "$URL" 2>/dev/null | head -c 200000 || true)
        if echo "$BODY" | grep -qiE '抽奖|lottery|🎁 娱乐'; then
            fail "面板响应中残留 lottery 字串"
        else
            pass "面板响应中未发现 lottery 字串"
        fi
    fi
fi

# ============================================================
# Section 9: xray 子进程
# ============================================================
if [[ $CHECK_RUNTIME -eq 1 ]]; then
    section "9. Runtime: xray 子进程"

    if pgrep -af 'xray-linux|xray\b' > /dev/null 2>&1; then
        pass "xray 子进程运行中"
        XPID=$(pgrep -f 'xray-linux' 2>/dev/null | head -1 || pgrep -f xray 2>/dev/null | head -1 || true)
        [[ -n "$XPID" ]] && info "xray PID: $XPID"
    else
        warn "未发现 xray 子进程（如未配置任何 inbound 这是正常的）"
    fi
fi

# ============================================================
# Section 10: TG 反遥测验证
# ============================================================
if [[ $CHECK_RUNTIME -eq 1 ]] && command -v ss >/dev/null 2>&1; then
    section "10. Runtime: TG 反遥测（采样 ${TG_SAMPLE_SEC}s）"

    info "观察 outbound TCP，检查是否有非用户配置的 t.me / api.telegram.org 连接..."
    info "（注意：Telegram 网段 149.154.160.0/20 + 91.108.4.0/22；如果你确实配了 TG bot 这是正常的）"

    TS_START=$(date +%s)
    TG_HITS=0
    SAMPLES=0
    while [[ $(($(date +%s) - TS_START)) -lt $TG_SAMPLE_SEC ]]; do
        SAMPLES=$((SAMPLES + 1))
        if ss -tn state established 2>/dev/null | tail -n +2 | awk '{print $4}' | \
           grep -qE '^(149\.154\.16[0-9]|149\.154\.17[0-9]|91\.108\.[4-7])\.'; then
            TG_HITS=$((TG_HITS + 1))
        fi
        sleep 2
    done

    if [[ $TG_HITS -eq 0 ]]; then
        pass "${TG_SAMPLE_SEC}s 内未观察到 Telegram 网段 outbound 连接"
    else
        warn "${TG_SAMPLE_SEC}s 内 ${TG_HITS}/${SAMPLES} 次采样命中 TG 网段（如已配 TG bot 属正常）"
    fi
fi

# ============================================================
# Section 11: 路线图 #14 主题文件
# ============================================================
section "11. ROADMAP #14: 5 主题文件"

# 5 主题：cyan / dark / blue / purple / orange（Stage 4 #14 commit 672b0a46）
THEME_DIR=""
for D in /usr/local/x-ui/web/assets/css /usr/local/x-ui/web/html/assets; do
    [[ -d "$D" ]] && THEME_DIR="$D" && break
done

if [[ -z "$THEME_DIR" ]]; then
    warn "未找到面板静态资源目录（panel 用 go embed 时可能没有外部文件）"
else
    info "静态资源目录: $THEME_DIR"
    # 不强求 5 个独立文件 —— 也可能在一个 css 里通过 CSS variable 切换；只标记找到的
    THEME_HITS=$(grep -rlE 'theme-(cyan|dark|blue|purple|orange)' "$THEME_DIR" 2>/dev/null | wc -l || echo 0)
    if [[ "${THEME_HITS:-0}" -gt 0 ]]; then
        pass "在 $THEME_HITS 个文件中发现 5 主题字串引用"
    else
        info "未在静态资源中发现主题字串（可能 embed 在二进制里）"
    fi
fi

# 二进制 strings 兜底验证
if [[ -e "$BIN_PATH" ]] && command -v strings >/dev/null 2>&1; then
    THEMES_IN_BIN=$(strings "$BIN_PATH" | grep -cE 'theme-(cyan|dark|blue|purple|orange)' || echo 0)
    if [[ "${THEMES_IN_BIN:-0}" -ge 3 ]]; then
        pass "二进制中嵌入的主题字串 ≥3 处（5 主题已构建到 binary）"
    else
        warn "二进制中主题字串 ${THEMES_IN_BIN:-0} 处（预期 ≥3）"
    fi
fi

# ============================================================
# 最终汇总
# ============================================================
section "12. 总结"

{
    echo ""
    echo "============================================"
    printf "总计: %d 项  |  PASS: %d  |  WARN: %d  |  FAIL: %d\n" \
        "$TOTAL" "$PASSED" "$WARNED" "$FAILED"
    echo "============================================"
    echo ""
    echo "完整报告写入: $REPORT"
    echo ""
    if [[ $FAILED -gt 0 ]]; then
        echo "[FAIL] 有 $FAILED 项 FAIL —— 详见报告，并比对 docs/VPS-VERIFICATION-CHECKLIST.md"
    elif [[ $WARNED -gt 0 ]]; then
        echo "[WARN] 无 FAIL 但有 $WARNED 项 WARN —— 多数与 'feature 未启用' 相关，可逐项确认"
    else
        echo "[ OK ] 全部通过"
    fi
} | tee -a "$REPORT"

# 上述 emoji 输出到 stdout/log，但脚本退出码以 FAILED 为准
[[ $FAILED -gt 0 ]] && exit 1
exit 0
