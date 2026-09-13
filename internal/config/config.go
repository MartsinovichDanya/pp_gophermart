package config

import (
	"flag"
	"log"
	"time"

	"github.com/caarlos0/env/v6"
)

// Config содержит настройки приложения.
type Config struct {
	RunAddress           string        `env:"RUN_ADDRESS" envDefault:"localhost:8080"`
	DatabaseURI          string        `env:"DATABASE_URI"`
	AccrualSystemAddress string        `env:"ACCRUAL_SYSTEM_ADDRESS"`
	LogLevel             string        `env:"LOG_LEVEL" envDefault:"DEBUG"`
	JWTSecret            string        `env:"JWT_SECRET" envDefault:"supersecretkey"`
	MaxBodySize          int           `env:"MAX_BODY_SIZE" envDefault:"2048"`
	ProcessingInterval   time.Duration `env:"PROCESSING_INTERVAL" envDefault:"5s"`
}

// GetConfig обрабатывает аргументы командной строки и переменные окружения,
// возвращает заполненную конфигурацию.
// Переменные окружения имеют приоритет над флагами.
func GetConfig() *Config {
	var cfg Config
	var flagRunAddress, flagDatabaseURI, flagAccrualAddress string

	// Парсим переменные окружения
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	// Определяем флаги командной строки
	flag.StringVar(&flagRunAddress, "a", "localhost:8080", "адрес запуска HTTP-сервера")
	flag.StringVar(&flagDatabaseURI, "d", "", "строка подключения к базе данных")
	flag.StringVar(&flagAccrualAddress, "r", "", "адрес системы расчёта начислений")
	flag.Parse()

	// Применяем флаги, если переменные окружения не заданы
	if cfg.RunAddress == "" {
		cfg.RunAddress = flagRunAddress
	}
	if cfg.DatabaseURI == "" {
		cfg.DatabaseURI = flagDatabaseURI
	}
	if cfg.AccrualSystemAddress == "" {
		cfg.AccrualSystemAddress = flagAccrualAddress
	}

	return &cfg
}
