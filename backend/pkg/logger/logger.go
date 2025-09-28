package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
)

type Logger struct {
	info          *log.Logger
	warning       *log.Logger
	error         *log.Logger
	structured    *StructuredLogger
	useStructured bool
}

func NewLogger() *Logger {
	// 检查是否启用结构化日志
	useStructured := strings.ToLower(os.Getenv("LOG_FORMAT")) == "json"

	return &Logger{
		info:          log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile),
		warning:       log.New(os.Stdout, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile),
		error:         log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile),
		structured:    NewStructuredLogger(useStructured),
		useStructured: useStructured,
	}
}

func (l *Logger) Info(v ...interface{}) {
	if l.useStructured {
		l.structured.Info(fmt.Sprint(v...))
	} else {
		l.info.Println(v...)
	}
}

func (l *Logger) Infof(format string, v ...interface{}) {
	if l.useStructured {
		l.structured.Info(fmt.Sprintf(format, v...))
	} else {
		l.info.Printf(format, v...)
	}
}

func (l *Logger) Warning(v ...interface{}) {
	if l.useStructured {
		l.structured.Warn(fmt.Sprint(v...))
	} else {
		l.warning.Println(v...)
	}
}

func (l *Logger) Warningf(format string, v ...interface{}) {
	if l.useStructured {
		l.structured.Warn(fmt.Sprintf(format, v...))
	} else {
		l.warning.Printf(format, v...)
	}
}

func (l *Logger) Error(v ...interface{}) {
	if l.useStructured {
		l.structured.Error(fmt.Sprint(v...), nil)
	} else {
		l.error.Println(v...)
	}
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	if l.useStructured {
		l.structured.Error(fmt.Sprintf(format, v...), nil)
	} else {
		l.error.Printf(format, v...)
	}
}

// 新增方法以支持结构化日志的高级功能
func (l *Logger) InfoWithFields(msg string, fields map[string]interface{}) {
	if l.useStructured {
		l.structured.Info(msg, fields)
	} else {
		l.info.Println(msg)
	}
}

func (l *Logger) ErrorWithErr(msg string, err error) {
	if l.useStructured {
		l.structured.Error(msg, err)
	} else {
		if err != nil {
			l.error.Printf("%s: %v", msg, err)
		} else {
			l.error.Println(msg)
		}
	}
}

func (l *Logger) WarnWithFields(msg string, fields map[string]interface{}) {
	if l.useStructured {
		l.structured.Warn(msg, fields)
	} else {
		l.warning.Println(msg)
	}
}

// GetStructuredLogger 获取结构化日志记录器
func (l *Logger) GetStructuredLogger() *StructuredLogger {
	return l.structured
}
