package locale

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pelletier/go-toml/v2"
	"golang.org/x/text/language"
)

// TestParseAllTranslationFiles 是 ce-1.0.1 教训留下的回归测试。
//
// 当时 [pages.inbounds.ceQuickDeploy] section 错位，造成 [pages.inbounds]
// 内部出现重复的 "title" key，go-toml/v2 strict 解析直接致 panel 启动 crash。
// 这个测试用与 production 路径 (locale.go:parseTranslationFiles)
// 一致的 i18n.Bundle.ParseMessageFileBytes 解析 web/translation/*.toml，
// 任何 toml 语法错 / 重复 key 都会在 CI / 本地 go test 阶段就暴露，
// 不再让 panel 启动崩溃成为发现问题的渠道。
func TestParseAllTranslationFiles(t *testing.T) {
	bundle := i18n.NewBundle(language.MustParse("en-US"))
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	tomls, err := filepath.Glob("../translation/*.toml")
	if err != nil {
		t.Fatalf("glob translation files: %v", err)
	}
	if len(tomls) == 0 {
		t.Fatal("no translation files found under ../translation/*.toml")
	}

	for _, path := range tomls {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if _, err := bundle.ParseMessageFileBytes(data, path); err != nil {
			t.Errorf("parse %s: %v", path, err)
		}
	}
}
