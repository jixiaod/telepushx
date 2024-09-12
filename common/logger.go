package common

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type DailyRotateWriter struct {
	dir     string
	prefix  string
	ext     string
	file    *os.File
	mu      sync.Mutex
	day     int
	writers []io.Writer
}

func NewDailyRotateWriter(dir, prefix, ext string, additionalWriters ...io.Writer) *DailyRotateWriter {
	w := &DailyRotateWriter{
		dir:     dir,
		prefix:  prefix,
		ext:     ext,
		writers: additionalWriters,
	}
	w.Rotate()
	return w
}

func (w *DailyRotateWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.day != time.Now().Day() {
		if err := w.Rotate(); err != nil {
			return 0, err
		}
	}

	n, err = w.file.Write(p)
	if err != nil {
		return n, err
	}

	for _, writer := range w.writers {
		if _, err := writer.Write(p); err != nil {
			return n, err
		}
	}

	return n, nil
}

func (w *DailyRotateWriter) Rotate() error {
	if w.file != nil {
		w.file.Close()
	}

	now := time.Now()
	w.day = now.Day()

	filename := filepath.Join(w.dir, fmt.Sprintf("%s%s%s", w.prefix, now.Format("2006-01-02"), w.ext))
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	w.file = file
	return nil
}

func SetupDailyRotateLog() {
	if *LogDir != "" {
		commonWriter := NewDailyRotateWriter(*LogDir, "info.", ".log", os.Stdout)
		errorWriter := NewDailyRotateWriter(*LogDir, "error.", ".log", os.Stderr)

		gin.DefaultWriter = commonWriter
		gin.DefaultErrorWriter = errorWriter
	}
}

func SetupGinLog() {
	if *LogDir != "" {
		commonLogPath := filepath.Join(*LogDir, "info.log")
		errorLogPath := filepath.Join(*LogDir, "error.log")
		commonFd, err := os.OpenFile(commonLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal("failed to open log file")
		}
		errorFd, err := os.OpenFile(errorLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal("failed to open log file")
		}
		gin.DefaultWriter = io.MultiWriter(os.Stdout, commonFd)
		gin.DefaultErrorWriter = io.MultiWriter(os.Stderr, errorFd)
	}
}

func SysLog(s string) {
	t := time.Now()
	_, _ = fmt.Fprintf(gin.DefaultWriter, "[SYS] %v | %s \n", t.Format("2006/01/02 - 15:04:05"), s)
}

func SysError(s string) {
	t := time.Now()
	_, _ = fmt.Fprintf(gin.DefaultErrorWriter, "[SYS] %v | %s \n", t.Format("2006/01/02 - 15:04:05"), s)
}

func FatalLog(v ...any) {
	t := time.Now()
	_, _ = fmt.Fprintf(gin.DefaultErrorWriter, "[FATAL] %v | %v \n", t.Format("2006/01/02 - 15:04:05"), v)
	os.Exit(1)
}
