package main

import "github.com/Ernesto-Che-Guevara/logrusplus"

// Импортируем твою библиотеку с алиасом logp

var logp = logrusplus.Log

func main() {
	// 1. Настраиваем логгер один раз при старте
	logrusplus.Init(logrusplus.LoggerConfig{
		ServiceName: "my_app",
		Mode:        logrusplus.ModeConsole,
	})

	// 2. Пользуемся ВСЕЙ мощью logrus через сущность Log!

	// Обычный лог
	logp.Info("Приложение запущено")

	// Форматированный лог (теперь он доступен из коробки!)
	port := 8080
	logp.Infof("Сервер слушает порт %d", port)

	// Продвинутые фичи logrus (например, прикрепление дополнительных полей)
	logp.WithFields(map[string]interface{}{
		"user_id": 42,
		"ip":      "192.168.1.1",
	}).Warn("Обнаружена подозрительная активность")
}
