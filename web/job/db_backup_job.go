package job

import (
	"x-ui/logger"
	"x-ui/web/service"
)

// CE 路线图 #100：自动 DB 备份 cron job。
//
// 由 web.go 注册为每日 03:00 触发，调用 service.DBBackupService.BackupOnce
// 在 <DBFolderPath>/backup/ 下生成 <name>-YYYYMMDD-HHMMSS.db.gz，
// 然后清理超过保留份数的旧备份。
//
// 不依赖外部 sqlite3 CLI，不向远程上报任何信息。
type DBBackupJob struct {
	backupService service.DBBackupService
}

func NewDBBackupJob() *DBBackupJob {
	return new(DBBackupJob)
}

func (j *DBBackupJob) Run() {
	path, err := j.backupService.BackupOnce()
	if err != nil {
		logger.Warning("CE #100: DB backup failed:", err)
		return
	}
	logger.Info("CE #100: DB backup created:", path)

	if err := j.backupService.CleanupOldBackups(); err != nil {
		logger.Warning("CE #100: DB backup cleanup failed:", err)
	}
}
