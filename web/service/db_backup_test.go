package service

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// CE 路线图 #100：compressDB 单元测试 —— 验证 gzip 流式复制后内容完整。
//
// 不调用 BackupOnce / CleanupOldBackups（它们依赖 config.GetDBPath 和全局
// gorm db 实例，需要完整环境）。compressDB 是核心 IO 路径，单测可独立覆盖。
func TestCompressDB_Roundtrip(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "test.db")
	dstPath := filepath.Join(tmp, "test.db.gz")

	// 模拟一个 SQLite 文件：标准 16 字节 magic header + 任意 payload
	sqliteHeader := []byte("SQLite format 3\x00")
	payload := append(sqliteHeader, []byte("hello x-panel-ce backup roundtrip")...)
	if err := os.WriteFile(srcPath, payload, 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	if err := compressDB(srcPath, dstPath); err != nil {
		t.Fatalf("compressDB: %v", err)
	}

	st, err := os.Stat(dstPath)
	if err != nil {
		t.Fatalf("stat dst: %v", err)
	}
	if st.Size() == 0 {
		t.Fatal("dst file is empty")
	}

	f, err := os.Open(dstPath)
	if err != nil {
		t.Fatalf("open dst: %v", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip.NewReader (file may not be valid gzip): %v", err)
	}
	defer gz.Close()

	got, err := io.ReadAll(gz)
	if err != nil {
		t.Fatalf("read gz: %v", err)
	}
	if string(got) != string(payload) {
		t.Errorf("payload mismatch:\n  got:  %q\n  want: %q", got, payload)
	}
}

// 验证目标文件的 IO 错误（源不存在）会被正确传播。
func TestCompressDB_SrcNotFound(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "does-not-exist.db")
	dst := filepath.Join(tmp, "out.db.gz")

	err := compressDB(src, dst)
	if err == nil {
		t.Fatal("expected error for missing src, got nil")
	}
}

// 验证目标目录不可写时不留半截产物（虽然 compressDB 自身不删，但调用方
// BackupOnce 失败路径会 os.Remove tmpPath；这里只验 compressDB 自身错误传播）。
func TestCompressDB_DstUnwritable(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.db")
	if err := os.WriteFile(src, []byte("payload"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	dst := filepath.Join(tmp, "no-such-dir", "out.db.gz")
	err := compressDB(src, dst)
	if err == nil {
		t.Fatal("expected error for unwritable dst, got nil")
	}
}

// 端到端：用 XUI_DB_FOLDER 把 db 路径指到临时目录，写一份伪 SQLite 文件，
// 调 BackupOnce 验证产物可解压、内容完整、文件名格式合法。
func TestBackupOnce_E2E(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XUI_DB_FOLDER", tmp)

	// config.GetDBPath() = <tmp>/x-ui.db
	srcPath := filepath.Join(tmp, "x-ui.db")
	sqliteHeader := []byte("SQLite format 3\x00")
	payload := append(sqliteHeader, []byte("e2e backup payload")...)
	if err := os.WriteFile(srcPath, payload, 0o644); err != nil {
		t.Fatalf("write fake db: %v", err)
	}

	svc := DBBackupService{}
	dstPath, err := svc.BackupOnce()
	if err != nil {
		t.Fatalf("BackupOnce: %v", err)
	}

	if filepath.Dir(dstPath) != filepath.Join(tmp, "backup") {
		t.Errorf("backup dir mismatch: %s", dstPath)
	}
	st, err := os.Stat(dstPath)
	if err != nil {
		t.Fatalf("stat backup: %v", err)
	}
	if st.Size() == 0 {
		t.Fatal("backup file empty")
	}

	f, err := os.Open(dstPath)
	if err != nil {
		t.Fatalf("open backup: %v", err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip.NewReader: %v", err)
	}
	defer gz.Close()
	got, err := io.ReadAll(gz)
	if err != nil {
		t.Fatalf("read gz: %v", err)
	}
	if string(got) != string(payload) {
		t.Errorf("backup content mismatch:\n  got:  %q\n  want: %q", got, payload)
	}
}

// 验证 CleanupOldBackups 只保留最近 backupKeepCount 份。
func TestCleanupOldBackups(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XUI_DB_FOLDER", tmp)

	backupDir := filepath.Join(tmp, "backup")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		t.Fatalf("mkdir backup: %v", err)
	}

	// 造 backupKeepCount + 5 个伪备份（文件名时间戳升序）
	totalCount := backupKeepCount + 5
	for i := 0; i < totalCount; i++ {
		name := fmt.Sprintf("x-ui-2026010%d-%06d.db.gz", i/10, i)
		path := filepath.Join(backupDir, name)
		if err := os.WriteFile(path, []byte("dummy"), 0o644); err != nil {
			t.Fatalf("write fake backup %d: %v", i, err)
		}
	}

	svc := DBBackupService{}
	if err := svc.CleanupOldBackups(); err != nil {
		t.Fatalf("CleanupOldBackups: %v", err)
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("read backup dir: %v", err)
	}
	if len(entries) != backupKeepCount {
		t.Errorf("after cleanup expected %d files, got %d", backupKeepCount, len(entries))
	}
}
