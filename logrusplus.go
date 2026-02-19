package logrusplus

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

// CustomFormatter — оставляем без изменений
type CustomFormatter struct {
	ServiceName   string
	DisableColors bool
}

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	// ... (здесь весь твой код метода Format, который мы писали ранее, он не меняется)
	// Для краткости я не дублирую его целиком, просто оставь его как есть!
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

	if entry.HasCaller() {
		line = entry.Caller.Line
		parts := strings.Split(entry.Caller.Function, "/")
		funcParts := strings.Split(parts[len(parts)-1], ".")
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

// --- НОВАЯ МАГИЯ НАЧИНАЕТСЯ ЗДЕСЬ ---

// logrusLogger — наш скрытый глобальный логгер с базовыми настройками
var logrusLogger = logrus.New()

// Init настраивает наш глобальный логгер
func Init(cfg LoggerConfig) {
	logrusLogger.SetReportCaller(true)

	formatter := &CustomFormatter{
		ServiceName: cfg.ServiceName,
	}

	// --- НОВАЯ ЛОГИКА: Подготовка пути и папки ---
	if cfg.Mode == ModeFile || cfg.Mode == ModeRolling {
		// Если путь не задан, генерируем стандартный
		if cfg.FilePath == "" {
			// Формат времени в Go задается через эталонную дату: 2006-01-02 15:04:05
			timestamp := time.Now().Format("2006-01-02_15-04-05")
			cfg.FilePath = fmt.Sprintf("./logs/%s.log", timestamp)
		}

		// Вытаскиваем путь к директории (например, из "./logs/2026-02-19.log" получим "./logs")
		dir := filepath.Dir(cfg.FilePath)

		// Создаем папку (0755 - стандартные права доступа: чтение/выполнение для всех, запись только для владельца)
		if err := os.MkdirAll(dir, 0755); err != nil {
			logrusLogger.Warnf("Не удалось создать папку для логов '%s': %v", dir, err)
			// Мы просто выводим предупреждение. Код пойдет дальше и, скорее всего,
			// упадет на создании файла, переключившись на консоль (как у нас уже прописано в ModeFile)
		}
	}
	// ----------------------------------------------

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

// Обертки для публичного использования

func Info(args ...interface{}) {
	logrusLogger.Info(args...)
}

func Error(args ...interface{}) {
	logrusLogger.Error(args...)
}

func Warn(args ...interface{}) {
	logrusLogger.Warn(args...)
}

func Debug(args ...interface{}) {
	logrusLogger.Debug(args...)
}
