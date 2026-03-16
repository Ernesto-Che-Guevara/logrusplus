package logrusplus

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	ModeConsole = "console"
	ModeFile    = "file"
	ModeRolling = "rolling"
)

type LoggerConfig struct {
	ServiceName string
	Mode        string
	FilePath    string
}

type CustomFormatter struct {
	ServiceName   string
	DisableColors bool
}

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	var levelColor, resetColor, levelText string
	switch entry.Level {
	case logrus.DebugLevel, logrus.TraceLevel:
		levelText = "DEBUG"
	case logrus.InfoLevel:
		levelText = "INFO"
	case logrus.WarnLevel:
		levelText = "WARN"
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		levelText = "ERROR"
	default:
		levelText = "UNKNOWN"
	}

	if !f.DisableColors {
		switch entry.Level {
		case logrus.DebugLevel, logrus.TraceLevel:
			levelColor = "\x1b[36m"
		case logrus.InfoLevel:
			levelColor = "\x1b[32m"
		case logrus.WarnLevel:
			levelColor = "\x1b[33m"
		case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
			levelColor = "\x1b[31m"
		}
		resetColor = "\x1b[0m"
	}

	timestamp := entry.Time.Format("2006.01.02 15:04:05")

	pkgName, className, funcName := "unknown", " ", "unknown"
	line := 0

	// Умный поиск по стеку вызовов
	var frame runtime.Frame
	pcs := make([]uintptr, 15)
	// Пропускаем первые уровни (внутренности Format и самого logrus)
	n := runtime.Callers(4, pcs)
	if n > 0 {
		frames := runtime.CallersFrames(pcs[:n])
		for {
			f, more := frames.Next()
			// Игнорируем стандартный logrus и нашу обертку logrusplus
			if !strings.Contains(f.Function, "sirupsen/logrus") && !strings.Contains(f.Function, "logrusplus") {
				frame = f
				break
			}
			if !more {
				break
			}
		}
	}

	if frame.PC != 0 {
		line = frame.Line
		fullFuncName := frame.Function

		parts := strings.Split(fullFuncName, "/")
		lastPart := parts[len(parts)-1]

		funcParts := strings.Split(lastPart, ".")
		if len(funcParts) >= 2 {
			pkgName = funcParts[0]
			if len(funcParts) == 3 {
				className = strings.Trim(funcParts[1], "()*")
				funcName = funcParts[2]
			} else {
				funcName = funcParts[1]
			}
		}
	}

	hierarchy := fmt.Sprintf("%s:%s:%s:%s:%d", f.ServiceName, pkgName, className, funcName, line)

	if f.DisableColors {
		fmt.Fprintf(b, "[%s] %s %s: %s\n", levelText, timestamp, hierarchy, entry.Message)
	} else {
		fmt.Fprintf(b, "%s[%s]%s %s %s: %s\n", levelColor, levelText, resetColor, timestamp, hierarchy, entry.Message)
	}
	return b.Bytes(), nil
}

// --- ПУБЛИЧНАЯ СУЩНОСТЬ ЛОГГЕРА ---

// Log открыт для использования, если нужны специфичные методы logrus (WithField и т.д.)
var Log = logrus.New()

func Init(cfg LoggerConfig) {
	Log.SetReportCaller(true)

	formatter := &CustomFormatter{
		ServiceName: cfg.ServiceName,
	}

	if cfg.Mode == ModeFile || cfg.Mode == ModeRolling {
		if cfg.FilePath == "" {
			timestamp := time.Now().Format("2006-01-02_15-04-05")
			cfg.FilePath = fmt.Sprintf("./logs/%s.log", timestamp)
		}

		dir := filepath.Dir(cfg.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			Log.Warnf("Не удалось создать папку для логов '%s': %v", dir, err)
		}
	}

	switch cfg.Mode {
	case ModeConsole:
		Log.SetOutput(os.Stdout)
		formatter.DisableColors = false

	case ModeFile:
		file, err := os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			Log.Warn("Не удалось открыть файл логов, используем консоль")
			Log.SetOutput(os.Stdout)
		} else {
			Log.SetOutput(file)
			formatter.DisableColors = true
		}

	case ModeRolling:
		rollingLogger := &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxAge:     1,
			MaxBackups: 30,
			MaxSize:    100,
			LocalTime:  true,
			Compress:   false,
		}
		Log.SetOutput(io.MultiWriter(rollingLogger))
		formatter.DisableColors = true
	}

	Log.SetFormatter(formatter)
}

// --- ПУБЛИЧНЫЕ ОБЕРТКИ ДЛЯ БЫСТРОГО ДОСТУПА ---

// 1. Базовые методы вывода
func Trace(args ...interface{})   { Log.Trace(args...) }
func Debug(args ...interface{})   { Log.Debug(args...) }
func Info(args ...interface{})    { Log.Info(args...) }
func Warn(args ...interface{})    { Log.Warn(args...) }
func Warning(args ...interface{}) { Log.Warning(args...) } // Warning - это синоним Warn в logrus
func Error(args ...interface{})   { Log.Error(args...) }
func Fatal(args ...interface{})   { Log.Fatal(args...) }
func Panic(args ...interface{})   { Log.Panic(args...) }

// 2. Методы вывода с форматированием (подстановка переменных)
func Tracef(format string, args ...interface{})   { Log.Tracef(format, args...) }
func Debugf(format string, args ...interface{})   { Log.Debugf(format, args...) }
func Infof(format string, args ...interface{})    { Log.Infof(format, args...) }
func Warnf(format string, args ...interface{})    { Log.Warnf(format, args...) }
func Warningf(format string, args ...interface{}) { Log.Warningf(format, args...) }
func Errorf(format string, args ...interface{})   { Log.Errorf(format, args...) }
func Fatalf(format string, args ...interface{})   { Log.Fatalf(format, args...) }
func Panicf(format string, args ...interface{})   { Log.Panicf(format, args...) }

// 3. Методы вывода с принудительным переносом строки
func Traceln(args ...interface{})   { Log.Traceln(args...) }
func Debugln(args ...interface{})   { Log.Debugln(args...) }
func Infoln(args ...interface{})    { Log.Infoln(args...) }
func Warnln(args ...interface{})    { Log.Warnln(args...) }
func Warningln(args ...interface{}) { Log.Warningln(args...) }
func Errorln(args ...interface{})   { Log.Errorln(args...) }
func Fatalln(args ...interface{})   { Log.Fatalln(args...) }
func Panicln(args ...interface{})   { Log.Panicln(args...) }

// 4. Совместимость со стандартным логгером Go (fmt/log)
func Print(args ...interface{})                 { Log.Print(args...) }
func Printf(format string, args ...interface{}) { Log.Printf(format, args...) }
func Println(args ...interface{})               { Log.Println(args...) }

// --- ОБЕРТКИ ДЛЯ РАБОТЫ С ПОЛЯМИ И КОНТЕКСТОМ ---
// Возвращают *logrus.Entry, что позволяет выстраивать цепочки:
// loga.WithField("key", "val").Info("msg")

func WithField(key string, value interface{}) *logrus.Entry {
	return Log.WithField(key, value)
}

func WithFields(fields logrus.Fields) *logrus.Entry {
	return Log.WithFields(fields)
}

func WithError(err error) *logrus.Entry {
	return Log.WithError(err)
}

func WithContext(ctx context.Context) *logrus.Entry {
	return Log.WithContext(ctx)
}

func WithTime(t time.Time) *logrus.Entry {
	return Log.WithTime(t)
}
