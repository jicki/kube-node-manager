package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// LogLevel 日志级别
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

var levelNames = []string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"}

func (l LogLevel) String() string {
	if int(l) < len(levelNames) {
		return levelNames[l]
	}
	return "UNKNOWN"
}

// StructuredLogger 结构化日志记录器
type StructuredLogger struct {
	level       LogLevel
	structured  bool
	serviceName string
	version     string
	hostname    string
}

// LogEntry 日志条目
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Service   string                 `json:"service"`
	Version   string                 `json:"version,omitempty"`
	Hostname  string                 `json:"hostname,omitempty"`
	Message   string                 `json:"message"`
	Caller    string                 `json:"caller,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Duration  string                 `json:"duration,omitempty"`
	TraceID   string                 `json:"trace_id,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
}

// NewStructuredLogger 创建新的结构化日志记录器
func NewStructuredLogger(structured bool) *StructuredLogger {
	hostname, _ := os.Hostname()
	version := getVersionForLogger()

	return &StructuredLogger{
		level:       DEBUG,
		structured:  structured,
		serviceName: "kube-node-manager",
		version:     version,
		hostname:    hostname,
	}
}

// SetLevel 设置日志级别
func (sl *StructuredLogger) SetLevel(level LogLevel) {
	sl.level = level
}

// Debug 记录调试信息
func (sl *StructuredLogger) Debug(message string, fields ...map[string]interface{}) {
	sl.log(DEBUG, message, "", fields...)
}

// Info 记录信息
func (sl *StructuredLogger) Info(message string, fields ...map[string]interface{}) {
	sl.log(INFO, message, "", fields...)
}

// Warn 记录警告
func (sl *StructuredLogger) Warn(message string, fields ...map[string]interface{}) {
	sl.log(WARN, message, "", fields...)
}

// Error 记录错误
func (sl *StructuredLogger) Error(message string, err error, fields ...map[string]interface{}) {
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}
	sl.log(ERROR, message, errorMsg, fields...)
}

// Fatal 记录致命错误并退出
func (sl *StructuredLogger) Fatal(message string, err error, fields ...map[string]interface{}) {
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}
	sl.log(FATAL, message, errorMsg, fields...)
	os.Exit(1)
}

// WithField 添加单个字段
func (sl *StructuredLogger) WithField(key string, value interface{}) *StructuredLogger {
	newLogger := *sl
	return &newLogger
}

// WithFields 添加多个字段
func (sl *StructuredLogger) WithFields(fields map[string]interface{}) *StructuredLogger {
	newLogger := *sl
	return &newLogger
}

// WithDuration 记录带执行时间的日志
func (sl *StructuredLogger) WithDuration(message string, duration time.Duration, fields ...map[string]interface{}) {
	entry := sl.createLogEntry(INFO, message, "")
	entry.Duration = duration.String()
	if len(fields) > 0 {
		entry.Fields = fields[0]
	}
	sl.output(entry)
}

// WithRequest 记录HTTP请求相关日志
func (sl *StructuredLogger) WithRequest(message string, method, path string, statusCode int, duration time.Duration, fields ...map[string]interface{}) {
	entry := sl.createLogEntry(INFO, message, "")
	entry.Duration = duration.String()

	requestFields := map[string]interface{}{
		"method":      method,
		"path":        path,
		"status_code": statusCode,
	}

	if len(fields) > 0 {
		for k, v := range fields[0] {
			requestFields[k] = v
		}
	}

	entry.Fields = requestFields
	sl.output(entry)
}

// log 核心日志记录方法
func (sl *StructuredLogger) log(level LogLevel, message, errorMsg string, fields ...map[string]interface{}) {
	if level < sl.level {
		return
	}

	entry := sl.createLogEntry(level, message, errorMsg)
	if len(fields) > 0 {
		entry.Fields = fields[0]
	}

	sl.output(entry)
}

// createLogEntry 创建日志条目
func (sl *StructuredLogger) createLogEntry(level LogLevel, message, errorMsg string) *LogEntry {
	entry := &LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     level.String(),
		Service:   sl.serviceName,
		Version:   sl.version,
		Hostname:  sl.hostname,
		Message:   message,
	}

	if errorMsg != "" {
		entry.Error = errorMsg
	}

	// 获取调用者信息
	if pc, file, line, ok := runtime.Caller(3); ok {
		if fn := runtime.FuncForPC(pc); fn != nil {
			entry.Caller = fmt.Sprintf("%s:%d %s", filepath.Base(file), line, fn.Name())
		}
	}

	return entry
}

// output 输出日志
func (sl *StructuredLogger) output(entry *LogEntry) {
	if sl.structured {
		// 结构化JSON输出
		if data, err := json.Marshal(entry); err == nil {
			fmt.Println(string(data))
		} else {
			// 如果JSON序列化失败，回退到普通格式
			log.Printf("[%s] %s: %s", entry.Level, entry.Service, entry.Message)
		}
	} else {
		// 普通格式输出（开发环境友好）
		timestamp := entry.Timestamp
		if parsed, err := time.Parse(time.RFC3339Nano, timestamp); err == nil {
			timestamp = parsed.Format("2006-01-02 15:04:05")
		}

		logMsg := fmt.Sprintf("[%s] %s %s", entry.Level, timestamp, entry.Message)

		if entry.Error != "" {
			logMsg += fmt.Sprintf(" | error: %s", entry.Error)
		}

		if entry.Duration != "" {
			logMsg += fmt.Sprintf(" | duration: %s", entry.Duration)
		}

		if entry.Fields != nil {
			if fieldsJSON, err := json.Marshal(entry.Fields); err == nil {
				logMsg += fmt.Sprintf(" | fields: %s", string(fieldsJSON))
			}
		}

		fmt.Println(logMsg)
	}
}

// getVersionForLogger 获取版本信息（用于日志）
func getVersionForLogger() string {
	data, err := os.ReadFile("VERSION")
	if err != nil {
		return "dev"
	}
	return strings.TrimSpace(string(data))
}

// RequestLogger 中间件用的请求日志记录器
type RequestLogger struct {
	logger *StructuredLogger
}

// NewRequestLogger 创建请求日志记录器
func NewRequestLogger(structured bool) *RequestLogger {
	return &RequestLogger{
		logger: NewStructuredLogger(structured),
	}
}

// LogRequest 记录HTTP请求
func (rl *RequestLogger) LogRequest(method, path, clientIP, userAgent string, statusCode int, duration time.Duration, bodySize int) {
	level := INFO
	if statusCode >= 400 {
		level = WARN
	}
	if statusCode >= 500 {
		level = ERROR
	}

	fields := map[string]interface{}{
		"method":     method,
		"path":       path,
		"status":     statusCode,
		"client_ip":  clientIP,
		"user_agent": userAgent,
		"body_size":  bodySize,
	}

	message := fmt.Sprintf("%s %s %d", method, path, statusCode)
	entry := rl.logger.createLogEntry(level, message, "")
	entry.Duration = duration.String()
	entry.Fields = fields

	rl.logger.output(entry)
}

// ApplicationLogger 应用日志接口
type ApplicationLogger interface {
	Debug(message string, fields ...map[string]interface{})
	Info(message string, fields ...map[string]interface{})
	Warn(message string, fields ...map[string]interface{})
	Error(message string, err error, fields ...map[string]interface{})
	Fatal(message string, err error, fields ...map[string]interface{})
	WithDuration(message string, duration time.Duration, fields ...map[string]interface{})
}
