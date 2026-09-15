package route

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"homeessentials/backend/internal/auth"
	"homeessentials/backend/internal/config"
	"homeessentials/backend/internal/controller"
	"homeessentials/backend/internal/database"
	"homeessentials/backend/internal/handler"
	"homeessentials/backend/internal/middleware"
	"homeessentials/backend/internal/repository"
)

func NewRouter(
	cfg config.Config,
	mongo *database.Mongo,
	tokens *auth.TokenIssuer,
	uploadHandler *handler.UploadHandler,
	orderHandler *handler.OrderHandler,
	paymentHandler *handler.PaymentHandler,
	orderRepo *repository.OrderRepository,
	productImages controller.ProductImageDeleter,
	log *slog.Logger,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(requestLogMiddleware(log))

	r.Use(middleware.CORS(cfg.CORSOrigins))

	health := handler.NewHealthHandler(mongo)
	r.GET("/health", health.Liveness)
	r.GET("/ready", health.Readiness)

	adminRepo := repository.NewAdminRepository(mongo.Database)
	authCtrl := controller.NewAuthController(adminRepo, tokens)
	authHandler := handler.NewAuthHandler(authCtrl)

	productRepo := repository.NewProductRepository(mongo.Database)
	productCtrl := controller.NewProductController(productRepo, orderRepo, productImages, log)
	productHandler := handler.NewProductHandler(productCtrl)

	admin := r.Group("/api/v1/admin")
	admin.POST("/login", authHandler.Login)

	protected := admin.Group("")
	protected.Use(middleware.RequireAdminRole(tokens))
	protected.POST("/products", productHandler.Create)
	protected.GET("/products", productHandler.List)
	protected.GET("/products/:id", productHandler.Get)
	protected.PUT("/products/:id", productHandler.Update)
	protected.DELETE("/products/:id", productHandler.Delete)
	protected.POST("/uploads", uploadHandler.UploadProductImage)
	protected.POST("/uploads/video", uploadHandler.UploadProductVideo)
	protected.GET("/orders", orderHandler.ListAdmin)
	protected.GET("/orders/:id", orderHandler.GetAdmin)
	protected.PATCH("/orders/:id/status", orderHandler.UpdateStatusAdmin)
	protected.DELETE("/orders/:id", orderHandler.DeleteAdmin)

	v1 := r.Group("/api/v1")
	v1.GET("/products", productHandler.ListPublic)
	v1.GET("/products/:id", productHandler.GetPublic)
	v1.POST("/orders", orderHandler.CreatePublic)
	v1.POST("/orders/track", paymentHandler.TrackOrder)
	v1.POST("/payments/initialize", paymentHandler.Initialize)
	v1.POST("/payments/abandon", paymentHandler.Abandon)
	v1.GET("/payments/verify", paymentHandler.Verify)

	webhooks := v1.Group("/webhooks")
	webhooks.POST("/paystack", handler.PaystackWebhookRawBody(), paymentHandler.PaystackWebhook)

	return r
}

func requestLogMiddleware(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		c.Next()
		if path == "/health" || path == "/ready" {
			return
		}
		log.Info("request",
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
		)
	}
}
