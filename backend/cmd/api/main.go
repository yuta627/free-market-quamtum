package main

import (
	"log"
	"os"

	"fleamarket-backend/internal/handler"
	"fleamarket-backend/internal/infrastructure"
	"fleamarket-backend/internal/infrastructure/persistence"
	"fleamarket-backend/internal/infrastructure/router"
	"fleamarket-backend/internal/usecase"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	db, err := infrastructure.NewDB()
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	// Add address columns to users table if not exist
	sqlDB, _ := db.DB()
	for _, col := range []string{"postal_code varchar(10)", "prefecture varchar(20)", "city varchar(100)", "address_line varchar(200)", "building varchar(200)"} {
		sqlDB.Exec("ALTER TABLE users ADD COLUMN IF NOT EXISTS " + col)
	}

	userRepo := persistence.NewUserRepository(db)
	productRepo := persistence.NewProductRepository(db)
	msgRepo := persistence.NewMessageRepository(db)
	likeRepo := persistence.NewLikeRepository(db)

	userH := handler.NewUserHandler(userRepo)
	authUC := usecase.NewAuthUsecase(userRepo)
	productUC := usecase.NewProductUsecase(productRepo)
	msgUC := usecase.NewMessageUsecase(msgRepo, productRepo)

	authH := handler.NewAuthHandler(authUC)
	msgH := handler.NewMessageHandler(msgUC)

	geminiClient, err := infrastructure.NewGeminiClient()
	if err != nil {
		log.Printf("warning: gemini client not available: %v", err)
	}
	aiUC := usecase.NewAIUsecase(geminiClient, productRepo)
	aiH := handler.NewAIHandler(aiUC)

	notifRepo := persistence.NewNotificationRepository(db)
	notifH := handler.NewNotificationHandler(notifRepo)

	stripeClient, err := infrastructure.NewStripeClient()
	if err != nil {
		log.Printf("warning: stripe client not available: %v", err)
	}
	paymentUC := usecase.NewPaymentUsecase(stripeClient, productRepo, notifRepo, userRepo)
	paymentH := handler.NewPaymentHandler(paymentUC)

	likeUC := usecase.NewLikeUsecase(likeRepo, productRepo)
	likeH := handler.NewLikeHandler(likeUC)

	recClient := infrastructure.NewRecommendationClient()
	recUC := usecase.NewRecommendationUsecase(recClient, productRepo)
	recH := handler.NewRecommendationHandler(recUC)

	qrngClient := infrastructure.NewQRNGClient()
	quantumH := handler.NewQuantumHandler(qrngClient)

	auctionRepo := persistence.NewAuctionRepository(db)
	auctionUC := usecase.NewAuctionUsecase(auctionRepo, productRepo, qrngClient)
	auctionH := handler.NewAuctionHandler(auctionUC)
	productH := handler.NewProductHandler(productUC, auctionRepo)

	r := router.New(authH, productH, msgH, aiH, paymentH, likeH, recH, quantumH, auctionH, notifH, userH)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("server listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
