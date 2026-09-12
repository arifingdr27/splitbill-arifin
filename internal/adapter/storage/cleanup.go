package storage

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

// StartLocalImageCleanup deletes files under basePath/images older than retention.
// Runs once immediately, then on interval until ctx is cancelled. VM storage only.
func StartLocalImageCleanup(ctx context.Context, basePath string, retentionDays, intervalHours int, log *logrus.Logger) {
	imagesDir := filepath.Join(basePath, "images")
	retention := time.Duration(retentionDays) * 24 * time.Hour
	interval := time.Duration(intervalHours) * time.Hour

	run := func() {
		deleted, err := purgeOlderThan(imagesDir, retention, log)
		if err != nil {
			log.WithError(err).WithField("dir", imagesDir).Error("storage cleanup failed")
			return
		}
		log.WithFields(logrus.Fields{
			"dir":            imagesDir,
			"retention_days": retentionDays,
			"deleted":        deleted,
		}).Info("storage cleanup finished")
	}

	log.WithFields(logrus.Fields{
		"dir":             imagesDir,
		"retention_days":  retentionDays,
		"interval_hours":  intervalHours,
	}).Info("storage cleanup worker started")

	run()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Info("storage cleanup worker stopped")
				return
			case <-ticker.C:
				run()
			}
		}
	}()
}

func purgeOlderThan(dir string, retention time.Duration, log *logrus.Logger) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	cutoff := time.Now().Add(-retention)
	deleted := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			log.WithError(err).WithField("file", entry.Name()).Warn("skip cleanup entry")
			continue
		}
		if !info.ModTime().Before(cutoff) {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if err := os.Remove(path); err != nil {
			log.WithError(err).WithField("file", path).Warn("failed to delete old file")
			continue
		}
		deleted++
		log.WithFields(logrus.Fields{
			"file":     path,
			"mod_time": info.ModTime().UTC().Format(time.RFC3339),
		}).Info("deleted old storage file")
	}
	return deleted, nil
}
