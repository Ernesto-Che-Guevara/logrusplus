package logrusplus

import (
	"bytes"
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

// CustomFormatter — наше правило оформления логов
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

	// 3. Вытаскиваем "честную" иерархию (кто реально вызвал лог)
	pkgName, className, funcName := "unknown", " ", "unknown"
	line := 0

	// Вручную ищем в стеке первый кадр, который не относится к этой библиотеке
	var frame runtime.Frame
	pcs := make([]uintptr, 10)
	n := runtime.Callers(3, pcs) // Пропускаем первые кадры (саму функцию Format и Log)
	if n > 0 {
		frames := runtime.CallersFrames(pcs[:n])
		for {
			f, more := frames.Next()
			// Если мы нашли функцию, которая НЕ в нашем пакете и НЕ в самом logrus
			if !strings.Contains(f.Function, "logrusplus") && !strings.Contains(f.Function, "sirupsen/logrus") {
				frame = f
				break
			}
			if !more {
				break
			}
		}
	}

	// Если нашли подходящий кадр — парсим его
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

// --- ИНИЦИАЛИЗАЦИЯ И НАСТРОЙКА ---

// logrusLogger — наш скрытый глобальный логгер с базовыми настройками
var logrusLogger = logrus.New()

// Init настраивает наш глобальный логгер
func Init(cfg LoggerConfig) {
	// Эта настройка все еще полезна для базовой работы logrus
	logrusLogger.SetReportCaller(true)

	formatter := &CustomFormatter{
		ServiceName: cfg.ServiceName,
	}

	// Подготовка пути и папки
	if cfg.Mode == ModeFile || cfg.Mode == ModeRolling {
		if cfg.FilePath == "" {
			timestamp := time.Now().Format("2006-01-02_15-04-05")
			cfg.FilePath = fmt.Sprintf("./logs/%s.log", timestamp)
		}

		dir := filepath.Dir(cfg.FilePath)

		if err := os.MkdirAll(dir, 0755); err != nil {
			logrusLogger.Warnf("Не удалось создать папку для логов '%s': %v", dir, err)
		}
	}

	switch cfg.Mode {
	case ModeConsole:
		logrusLogger.SetOutput(os.Stdout)
		formatter.DisableColors = false

	case ModeFile:
		file, err := os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			logrusLogger.Warn("Не удалось открыть файл логов, используем консоль")
			logrusLogger.SetOutput(os.Stdout)
		} else {
			logrusLogger.SetOutput(file)
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
		logrusLogger.SetOutput(io.MultiWriter(rollingLogger))
		formatter.DisableColors = true
	}

	logrusLogger.SetFormatter(formatter)
}

// --- ПУБЛИЧНЫЕ МЕТОДЫ (ОБЕРТКИ) ---

func Info(args ...interface{}) {
	logrusLogger.Log(logrus.InfoLevel, args...)
}

func Error(args ...interface{}) {
	logrusLogger.Log(logrus.ErrorLevel, args...)
}

func Debug(args ...interface{}) {
	logrusLogger.Log(logrus.DebugLevel, args...)
}

func Warn(args ...interface{}) {
	logrusLogger.Log(logrus.WarnLevel, args...)
}
