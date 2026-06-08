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
	keywords := classifier.NewKeywordMatcher()
	llm := classifier.NewLLMClassifier()
	service := classifier.NewService(keywords, llm, policyStore, auditStore)

	detectHandler := handlers.NewDetectHandler(service)
	protectHandler := handlers.NewProtectHandler(service)
	policyHandler := handlers.NewPolicyHandler(policyStore)
	auditHandler := handlers.NewAuditHandler(auditStore)

	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery(), gin.Logger())

	engine.POST("/detect", detectHandler.Detect)
	engine.POST("/protect", protectHandler.Protect)

	engine.POST("/policies", policyHandler.Create)
	engine.GET("/policies", policyHandler.List)
	engine.GET("/policies/:id", policyHandler.Get)
	engine.PUT("/policies/:id", policyHandler.Update)
	engine.DELETE("/policies/:id", policyHandler.Delete)
	engine.GET("/audit", auditHandler.Query)

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("detection service listening on %s", addr)
	if err := engine.Run(addr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
