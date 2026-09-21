package config

import (
	"flag"
	"log"
	"time"

	"github.com/caarlos0/env/v6"
)

// Config содержит настройки приложения.
type Config struct {
	RunAddress           string        `env:"RUN_ADDRESS"`
	DatabaseURI          string        `env:"DATABASE_URI"`
	AccrualSystemAddress string        `env:"ACCRUAL_SYSTEM_ADDRESS"`
	LogLevel             string        `env:"LOG_LEVEL" envDefault:"DEBUG"`
	JWTSecret            string        `env:"JWT_SECRET" envDefault:"supersecretkey"`
	MaxBodySize          int           `env:"MAX_BODY_SIZE" envDefault:"2048"`
	ProcessingInterval   time.Duration `env:"PROCESSING_INTERVAL" envDefault:"5s"`
}

// GetConfig обрабатывает аргументы командной строки и переменные окружения,
// возвращает заполненную конфигурацию.
// Приоритет: переменные окружения > флаги командной строки > значения по умолчанию.
func GetConfig() *Config {
	var cfg Config

	// Определяем флаги командной строки с дефолтами
	flag.StringVar(&cfg.RunAddress, "a", "localhost:8080", "адрес запуска HTTP-сервера")
	flag.StringVar(&cfg.DatabaseURI, "d", "", "строка подключения к базе данных")
	flag.StringVar(&cfg.AccrualSystemAddress, "r", "", "адрес системы расчёта начислений")
	flag.Parse()

	// Парсим переменные окружения (перекроют флаги, если заданы)
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	return &cfg
}
