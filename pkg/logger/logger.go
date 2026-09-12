package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

func New(level, logDir string) (*logrus.Logger, error) {
	if err := os.MkdirAll(filepath.Join(logDir, "general_log"), 0o755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	date := time.Now().Format("01-02-2006")
	fileLogger := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, "general_log", date+".log"),
		MaxSize:    100,
		MaxBackups: 7,
		MaxAge:     30,
		Compress:   true,
		LocalTime:  true,
	}

	log := logrus.New()
	log.SetOutput(io.MultiWriter(os.Stdout, fileLogger))
	log.SetFormatter(&logrus.JSONFormatter{DisableHTMLEscape: true})
	log.SetReportCaller(true)

	lvl, err := logrus.ParseLevel(strings.ToLower(level))
	if err != nil {
		lvl = logrus.InfoLevel
	}
	log.SetLevel(lvl)
	return log, nil
}
