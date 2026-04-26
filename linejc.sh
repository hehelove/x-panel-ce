#!/usr/bin/env bash
# linejc.sh — x-panel-ce 线路 / IP 质量检测
#
# 路线图 #8（docs/ROADMAP.md §3.1）
# 数据源原则：仅使用以下公开免费端点和本地工具，不接入任何商业测速 API
#   - https://www.cloudflare.com/cdn-cgi/trace（公共，免费）
#   - https://ifconfig.io（公共，免费）
#   - https://ipinfo.io/<ip>/json（匿名级公共端点）
#   - 本地 ping / mtr / traceroute / /dev/tcp
#
# 许可证：GPL-3.0
# 仓库：https://github.com/hehelove/x-panel-ce

set -u
set -o pipefail

# ---------------- color ----------------
if [ -t 1 ] && [ "${TERM:-}" != "dumb" ]; then
    GREEN=$'\033[0;32m'
    YELLOW=$'\033[0;33m'
    RED=$'\033[0;31m'
    PLAIN=$'\033[0m'
else
    GREEN=""; YELLOW=""; RED=""; PLAIN=""
fi

require_cmd() {
    command -v "$1" >/dev/null 2>&1 || {
        echo "${RED}缺少依赖：$1，请先安装${PLAIN}" >&2
        exit 1
    }
}

require_cmd curl
require_cmd ping

echo ""
echo "${GREEN}========================================================${PLAIN}"
echo "${GREEN}  〔x-panel-ce〕线路 / IP 质量检测（Community Edition）${PLAIN}"
echo "${GREEN}========================================================${PLAIN}"
echo "${YELLOW}  数据源：Cloudflare cdn-cgi/trace + ipinfo.io + 本地 mtr/traceroute${PLAIN}"
echo "${YELLOW}  原则：不接入任何商业测速 API${PLAIN}"
echo ""

# ---------------- [1] 公网 IPv4 / IPv6 ----------------
echo "${GREEN}[1] 公网 IP${PLAIN}"
ipv4=$(curl -fsSL --max-time 5 -4 https://ifconfig.io 2>/dev/null || echo "N/A")
ipv6=$(curl -fsSL --max-time 5 -6 https://ifconfig.io 2>/dev/null || echo "N/A")
echo "  IPv4: ${ipv4}"
echo "  IPv6: ${ipv6}"
echo ""

# ---------------- [2] ipinfo.io 元数据 ----------------
echo "${GREEN}[2] IP 元数据（ipinfo.io）${PLAIN}"
if [ "${ipv4}" != "N/A" ]; then
    info=$(curl -fsSL --max-time 5 "https://ipinfo.io/${ipv4}/json" 2>/dev/null || echo "")
    if [ -n "${info}" ]; then
        for k in country region city org timezone; do
            v=$(echo "${info}" | grep -oE "\"${k}\": *\"[^\"]*\"" | sed -E 's/.*: *"([^"]*)"/\1/')
            [ -n "${v}" ] && printf "  %-9s %s\n" "${k}:" "${v}"
        done
    else
        echo "  ${YELLOW}查询失败（可能被限流，过几分钟再试）${PLAIN}"
    fi
else
    echo "  ${YELLOW}IPv4 未知，跳过${PLAIN}"
fi
echo ""

# ---------------- [3] Cloudflare 边缘节点 ----------------
echo "${GREEN}[3] Cloudflare cdn-cgi/trace${PLAIN}"
trace=$(curl -fsSL --max-time 5 https://www.cloudflare.com/cdn-cgi/trace 2>/dev/null || echo "")
if [ -n "${trace}" ]; then
    echo "${trace}" | grep -E "^(loc|colo|warp|ip|http)=" | sed 's/^/  /'
else
    echo "  ${YELLOW}查询失败${PLAIN}"
fi
echo ""

# ---------------- [4] 关键节点延迟 / 丢包 ----------------
echo "${GREEN}[4] 关键节点 ping（5 包，2 秒超时）${PLAIN}"
for host in 1.1.1.1 8.8.8.8 9.9.9.9 223.5.5.5; do
    out=$(ping -c 5 -W 2 "${host}" 2>/dev/null | tail -2)
    if [ -z "${out}" ]; then
        printf "  %-14s %s\n" "${host}" "${RED}不可达${PLAIN}"
        continue
    fi
    loss=$(echo "${out}" | grep -oE "[0-9]+%[^,]*" | head -n1)
    avg=$(echo "${out}" | grep -oE "[0-9]+\.[0-9]+/[0-9]+\.[0-9]+/[0-9]+\.[0-9]+" | head -n1 | awk -F'/' '{print $2}')
    printf "  %-14s avg=%sms loss=%s\n" "${host}" "${avg:-?}" "${loss:-?}"
done
echo ""

# ---------------- [5] 出方向 TCP 连通性 ----------------
echo "${GREEN}[5] 出方向 TCP 端口（80 / 443 / 22）${PLAIN}"
for target in "1.1.1.1:80" "1.1.1.1:443" "github.com:22"; do
    host="${target%:*}"
    port="${target##*:}"
    if timeout 3 bash -c ">/dev/tcp/${host}/${port}" 2>/dev/null; then
        printf "  %-18s %s\n" "${target}" "${GREEN}OK${PLAIN}"
    else
        printf "  %-18s %s\n" "${target}" "${RED}阻塞或不可达${PLAIN}"
    fi
done
echo ""

# ---------------- [6] mtr / traceroute 路由跳数 ----------------
if command -v mtr >/dev/null 2>&1; then
    echo "${GREEN}[6] mtr to 1.1.1.1（10 跳，3 包）${PLAIN}"
    mtr -nrwc 3 -m 10 1.1.1.1 2>/dev/null | sed 's/^/  /'
elif command -v traceroute >/dev/null 2>&1; then
    echo "${YELLOW}[6] mtr 不可用，回退 traceroute${PLAIN}"
    traceroute -n -w 1 -m 10 1.1.1.1 2>/dev/null | sed 's/^/  /'
else
    echo "${YELLOW}[6] 未安装 mtr / traceroute，跳过路由跳数检测${PLAIN}"
    echo "${YELLOW}    Debian/Ubuntu: apt install -y mtr-tiny${PLAIN}"
    echo "${YELLOW}    RHEL/CentOS:   yum install -y mtr${PLAIN}"
fi
echo ""

echo "${GREEN}========================================================${PLAIN}"
echo "${GREEN}  检测完成${PLAIN}"
echo "${GREEN}========================================================${PLAIN}"
echo ""
