package service

import (
	"strings"
	"testing"
)

// TestFormatUptimeSeconds 验证 CE #104 健康度报告的 uptime 字符串渲染。
func TestFormatUptimeSeconds(t *testing.T) {
	tests := []struct {
		secs uint64
		want string
	}{
		{0, "0m"},
		{59, "0m"},
		{60, "1m"},
		{3599, "59m"},
		{3600, "1h 0m"},
		{3601, "1h 0m"},
		{3661, "1h 1m"},
		{86400, "1d 0h 0m"},
		{86461, "1d 0h 1m"},
		{86400*5 + 3600*7 + 60*23, "5d 7h 23m"},
	}
	for _, tc := range tests {
		got := formatUptimeSeconds(tc.secs)
		if got != tc.want {
			t.Errorf("formatUptimeSeconds(%d) = %q, want %q", tc.secs, got, tc.want)
		}
	}
}

// TestPercentString 边界 + 除零保护。
func TestPercentString(t *testing.T) {
	tests := []struct {
		used, total uint64
		want        string
	}{
		{0, 100, "0.0%"},
		{50, 100, "50.0%"},
		{100, 100, "100.0%"},
		{12345, 0, "n/a"}, // 除零保护
		{0, 0, "n/a"},
		{1, 3, "33.3%"},
	}
	for _, tc := range tests {
		got := percentString(tc.used, tc.total)
		if got != tc.want {
			t.Errorf("percentString(%d, %d) = %q, want %q", tc.used, tc.total, got, tc.want)
		}
	}
}

// TestTrimMultiline 验证：
// - 多行压成单行（\n -> " | "）
// - 长度限制按 rune 切（中文不被截成乱码）
// - 头尾 trim 空白
func TestTrimMultiline(t *testing.T) {
	t.Run("multiline collapse", func(t *testing.T) {
		got := trimMultiline("line1\nline2\nline3", 0)
		if got != "line1 | line2 | line3" {
			t.Errorf("got %q", got)
		}
	})
	t.Run("trim outer whitespace", func(t *testing.T) {
		got := trimMultiline("\n\n  hello world  \n", 0)
		if got != "hello world" {
			t.Errorf("got %q", got)
		}
	})
	t.Run("rune-safe truncation", func(t *testing.T) {
		// 10 个中文字 + 限长 5 → 5 个汉字 + …
		input := "一二三四五六七八九十"
		got := trimMultiline(input, 5)
		runes := []rune(got)
		if len(runes) != 6 || runes[len(runes)-1] != '…' {
			t.Errorf("rune truncation broken: got %q (runes=%d)", got, len(runes))
		}
		if !strings.HasPrefix(got, "一二三四五") {
			t.Errorf("expected prefix 一二三四五, got %q", got)
		}
	})
	t.Run("zero maxLen no truncation", func(t *testing.T) {
		input := strings.Repeat("a", 1000)
		got := trimMultiline(input, 0)
		if got != input {
			t.Errorf("maxLen=0 should not truncate")
		}
	})
}
