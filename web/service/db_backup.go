package service

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"x-ui/config"
	"x-ui/database"
	"x-ui/logger"
)

// CE 路线图 #100：自动 DB 备份服务（仅本地，无远程上报）。
//
// 设计说明：
//   - 备份目录：<DBFolderPath>/backup/  （通常 /etc/x-ui/backup/）
//   - 文件命名：<name>-YYYYMMDD-HHMMSS.db.gz
//   - 流程：database.Checkpoint() → 复制 db 流式 gzip → 写临时文件 → 原子 rename
//   - 保留策略：保留最近 backupKeepCount 份，文件名时间戳升序删旧
//   - 并发保护：sync.Mutex 防 cron 与启动钩子同时执行
//   - 不调用外部 sqlite3 / cp 等系统命令，零运行时依赖
//
// 调用方：
//   - 每天 03:00 cron job (web/job/db_backup_job.go, 由 web/web.go 注册)
//   - 启动后异步延迟 30s 立即触发一次（应对长时间未重启场景）
type DBBackupService struct{}

const (
	// 保留最近 N 份备份。设为 14 应对每日 03:00 cron + 偶尔重启场景。
	backupKeepCount = 14
)

// backupMu 全局串行化备份操作，避免 cron 与启动钩子同时跑造成临时文件冲突。
var backupMu sync.Mutex

// BackupOnce 执行一次备份，成功返回备份文件绝对路径。
//
// timing/threading：
//   - 全局 mutex 串行化（同一进程内绝不并发执行）
//   - WAL checkpoint 失败仅警告，备份继续（极少数 db 未完成初始化场景）
//   - 失败会清理临时文件，不留半截产物
func (s *DBBackupService) BackupOnce() (string, error) {
	backupMu.Lock()
	defer backupMu.Unlock()

	srcPath := config.GetDBPath()
	if _, err := os.Stat(srcPath); err != nil {
		return "", fmt.Errorf("db file not found: %w", err)
	}

	// Checkpoint 之前先确认 db 已初始化，否则 database.Checkpoint() 内部会
	// 对 nil 的 *gorm.DB 解引用 panic。生产路径中 db 必定已 InitDB，但单测
	// 直接调 BackupOnce 会触发；这里做防御性检查。
	if database.GetDB() != nil {
		if err := database.Checkpoint(); err != nil {
			logger.Warning("CE #100: WAL checkpoint before backup failed (continuing):", err)
		}
	}

	backupDir := filepath.Join(config.GetDBFolderPath(), "backup")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir backup dir: %w", err)
	}

	ts := time.Now().Format("20060102-150405")
	dstName := fmt.Sprintf("%s-%s.db.gz", config.GetName(), ts)
	dstPath := filepath.Join(backupDir, dstName)
	tmpPath := dstPath + ".tmp"

	if err := compressDB(srcPath, tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}

	if err := os.Rename(tmpPath, dstPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("rename backup file: %w", err)
	}

	return dstPath, nil
}

// CleanupOldBackups 清理超过保留份数的旧备份。
// 不获取 backupMu：清理操作只读 + 删除已 rename 完成的成品文件，与备份本身互不冲突。
func (s *DBBackupService) CleanupOldBackups() error {
	backupDir := filepath.Join(config.GetDBFolderPath(), "backup")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	prefix := config.GetName() + "-"
	var backups []os.DirEntry
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !strings.HasPrefix(n, prefix) || !strings.HasSuffix(n, ".db.gz") {
			continue
		}
		backups = append(backups, e)
	}

	if len(backups) <= backupKeepCount {
		return nil
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Name() < backups[j].Name()
	})

	toDelete := len(backups) - backupKeepCount
	for i := 0; i < toDelete; i++ {
		path := filepath.Join(backupDir, backups[i].Name())
		if err := os.Remove(path); err != nil {
			logger.Warning("CE #100: failed to remove old backup:", path, "-", err)
		}
	}
	return nil
}

// compressDB 把 srcPath 的内容 gzip 压缩写入 dstPath。
// 单独抽出便于单测；不做 mutex 控制（由调用方保证）。
func compressDB(srcPath, dstPath string) error {
	in, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer in.Close()

	out, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return fmt.Errorf("create dst: %w", err)
	}
	defer out.Close()

	gz := gzip.NewWriter(out)
	if _, err := io.Copy(gz, in); err != nil {
		gz.Close()
		return fmt.Errorf("copy+gzip: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("close gzip: %w", err)
	}
	return out.Sync()
}
