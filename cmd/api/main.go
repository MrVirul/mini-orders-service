package main

import (
	"log"

	"github.com/MrVirul/mini-orders-service/internal/config"
	"github.com/MrVirul/mini-orders-service/internal/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := database.Connect(cfg.MasterDSN)
	if err != nil {
		log.Fatalf("connect to master database: %v", err)
	}
	defer db.Close()

	log.Println("connected to master database")
}
