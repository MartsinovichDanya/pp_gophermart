package main

import (
	"log"

	"github.com/MartsinovichDanya/pp_gophermart/internal/server"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
