package main

import (
	"log"

	"github.com/MartsinovichDanya/pp_gophermart/internal/server"
)

func main() {
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
