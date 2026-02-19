package main

import "github.com/Ernesto-Che-Guevara/logrusplus"

type MyDatabase struct {
	// Больше никакого logger *logrus.Logger здесь не нужно! 🎉
}

func (db *MyDatabase) Connect() {
	// Просто вызываем глобальную функцию из нашего пакета
	logrusplus.Info("started...")
	logrusplus.Error("connection error!")
}

func main() {
	// 1. Один раз при старте приложения инициализируем настройки
	cfg := logrusplus.LoggerConfig{
		ServiceName: "my_cool_app",
		Mode:        logrusplus.ModeConsole,
	}
	logrusplus.Init(cfg)

	// 2. Пользуемся логгером где угодно!
	logrusplus.Info("App is booting up")

	db := &MyDatabase{}
	db.Connect()
}
