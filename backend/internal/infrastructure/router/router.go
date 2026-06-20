package router

import (
	"fleamarket-backend/internal/handler"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func New(authH *handler.AuthHandler, productH *handler.ProductHandler, msgH *handler.MessageHandler, aiH *handler.AIHandler, paymentH *handler.PaymentHandler, likeH *handler.LikeHandler, recH *handler.RecommendationHandler, quantumH *handler.QuantumHandler, auctionH *handler.AuctionHandler, notifH *handler.NotificationHandler, userH *handler.UserHandler) *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:5173",
			"https://free-market-quamtum.vercel.app",
			"https://free-market-quamtum-*.vercel.app",
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/signup", authH.Signup)
			auth.POST("/login", authH.Login)
		}

		ai := api.Group("/ai")
		ai.Use(handler.AuthMiddleware())
		{
			ai.POST("/generate-description", aiH.GenerateDescription)
		}

		products := api.Group("/products")
		{
			products.GET("", productH.List)
			products.GET("/:id", productH.GetByID)
			products.GET("/:id/auction", productH.GetAuction)
			products.GET("/:id/recommendations", recH.GetRecommendations)
			products.GET("/:id/recommendations/qml", recH.GetQMLRecommendations)
			products.GET("/:id/recommendations/classical", recH.GetClassicalRecommendations)
			products.GET("/:id/recommendations/qkernel", recH.GetQKernelRecommendations)
			products.POST("/:id/ask", aiH.AskQuestion)

			authed := products.Group("")
			authed.Use(handler.AuthMiddleware())
			{
				authed.POST("", productH.Create)
				authed.POST("/:id/purchase", productH.Purchase)
				authed.POST("/:id/checkout", paymentH.CreateCheckout)
				authed.POST("/:id/confirm-purchase", paymentH.ConfirmPurchase)
				authed.POST("/:id/like", likeH.ToggleLike)
				authed.GET("/:id/messages", msgH.List)
				authed.POST("/:id/messages", msgH.Send)
			}
		}

		likes := api.Group("/likes")
		likes.Use(handler.AuthMiddleware())
		{
			likes.GET("", likeH.ListLikes)
		}

		me := api.Group("/me")
		me.Use(handler.AuthMiddleware())
		{
			me.GET("", userH.GetMe)
			me.GET("/products", productH.ListMine)
			me.GET("/purchases", productH.ListPurchased)
			me.PATCH("/address", userH.UpdateAddress)
		}

		notifications := api.Group("/notifications")
		notifications.Use(handler.AuthMiddleware())
		{
			notifications.GET("", notifH.List)
			notifications.PATCH("/:id/read", notifH.MarkRead)
			notifications.PATCH("/read-all", notifH.MarkAllRead)
		}

		quantum := api.Group("/quantum")
		{
			quantum.GET("/random", quantumH.GetRandom)
		}

		auctions := api.Group("/auctions")
		{
			auctions.GET("", auctionH.List)
			auctions.GET("/:id", auctionH.GetByID)
			authedAuctions := auctions.Group("")
			authedAuctions.Use(handler.AuthMiddleware())
			{
				authedAuctions.POST("", auctionH.Create)
				authedAuctions.POST("/:id/bid", auctionH.PlaceBid)
				authedAuctions.POST("/:id/finalize", auctionH.Finalize)
			}
		}
	}

	return r
}
