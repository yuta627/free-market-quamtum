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
	// Run schema migrations
	sqlDB, _ := db.DB()
	migrations := []string{
		// users: address columns
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS postal_code varchar(10)",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS prefecture varchar(20)",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS city varchar(100)",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS address_line varchar(200)",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS building varchar(200)",
		// likes table
		`CREATE TABLE IF NOT EXISTS likes (
			id         BIGSERIAL   PRIMARY KEY,
			user_id    BIGINT      NOT NULL REFERENCES users(id),
			product_id BIGINT      NOT NULL REFERENCES products(id),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_likes_user_product ON likes(user_id, product_id)",
		"CREATE INDEX IF NOT EXISTS idx_likes_product_id ON likes(product_id)",
		"ALTER TABLE likes ADD COLUMN IF NOT EXISTS liked BOOLEAN NOT NULL DEFAULT TRUE",
		"ALTER TABLE likes ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()",
		// messages table
		`CREATE TABLE IF NOT EXISTS messages (
			id         BIGSERIAL   PRIMARY KEY,
			product_id BIGINT      NOT NULL REFERENCES products(id),
			sender_id  BIGINT      NOT NULL REFERENCES users(id),
			body       TEXT        NOT NULL,
			is_read    BOOLEAN     NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			deleted_at TIMESTAMPTZ
		)`,
		"CREATE INDEX IF NOT EXISTS idx_messages_product_id ON messages(product_id)",
		"CREATE INDEX IF NOT EXISTS idx_messages_sender_id ON messages(sender_id)",
	}
	for _, m := range migrations {
		if _, err := sqlDB.Exec(m); err != nil {
			log.Printf("migration warning: %v", err)
		}
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
