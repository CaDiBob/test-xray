package API

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)


var (
	apiLogDir   = getenv("API_LOG_DIR", "/var/log/xray")
	apiInfoName = getenv("API_INFO_LOG", "api-info.log")
	apiErrName  = getenv("API_ERROR_LOG", "api-error.log")


	initOnce      sync.Once
	infoLogger    *log.Logger
	errLogger     *log.Logger
	infoCloser    io.Closer
	errCloser     io.Closer
)

// ensureInit открывает файлы и настраивает log.Logger один раз.
func ensureInit() {
	initOnce.Do(func() {
		_ = os.MkdirAll(apiLogDir, 0o755)

		infoPath := filepath.Join(apiLogDir, apiInfoName)
		errPath := filepath.Join(apiLogDir, apiErrName)

		infoFile, err := os.OpenFile(infoPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			infoLogger = log.New(os.Stdout, "[API][INFO] ", log.Ldate|log.Ltime|log.Lmicroseconds)
		} else {
			infoCloser = infoFile
			var w io.Writer = infoFile
			if alsoStdout {
				w = io.MultiWriter(w, os.Stdout)
			}
			infoLogger = log.New(w, "[API][INFO] ", log.Ldate|log.Ltime|log.Lmicroseconds)
		}

		errFile, e2 := os.OpenFile(errPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if e2 != nil {
			errLogger = log.New(os.Stderr, "[API][ERROR] ", log.Ldate|log.Ltime|log.Lmicroseconds)
		} else {
			errCloser = errFile
			var w io.Writer = errFile
			if alsoErrors {
				w = io.MultiWriter(w, os.Stderr, errFile)
			}
			errLogger = log.New(w, "[API][ERROR] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
		}
	})
}


func closeAPILoggers() {
	if infoCloser != nil {
		_ = infoCloser.Close()
	}
	if errCloser != nil {
		_ = errCloser.Close()
	}
}


func APIInfof(format string, args ...any) {
	ensureInit()
	infoLogger.Printf(format, args...)
}

func APIDebugf(format string, args ...any) {
	if getenv("API_DEBUG", "") == "" {
		return
	}
	ensureInit()
	infoLogger.Printf("[DEBUG] "+format, args...)
}

func APIErrorf(format string, args ...any) {
	ensureInit()
	errLogger.Printf(format, args...)
}

func APIWrapErr(prefix string, err error) error {
	if err == nil {
		return nil
	}
	wrapped := fmt.Errorf("%s: %w", prefix, err)
	APIErrorf("%s: %v", prefix, err)
	return wrapped
}

func kv(k string, v any) string {
	return fmt.Sprintf("%s=%v", k, v)
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
