package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"detection-service/internal/classifier"
	"detection-service/internal/handlers"
	"detection-service/internal/store"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "detection.db"
	}

	db, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	policyStore := store.NewPolicyStore(db)
	if err := policyStore.SeedDefault(); err != nil {
		log.Fatalf("failed to seed default policy: %v", err)
	}

	auditStore := store.NewAuditStore(db)
	llm := classifier.NewLLMClassifier()
	service := classifier.NewService(llm, policyStore, auditStore)
	detectHandler := handlers.NewDetectHandler(service)

	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery(), gin.Logger())

	engine.POST("/detect", detectHandler.Detect)

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("detection service listening on %s", addr)
	if err := engine.Run(addr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
