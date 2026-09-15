package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"homeessentials/backend/internal/auth"
	"homeessentials/backend/internal/config"
	"homeessentials/backend/internal/controller"
	"homeessentials/backend/internal/database"
	"homeessentials/backend/internal/handler"
	"homeessentials/backend/internal/jobs"
	"homeessentials/backend/internal/logging"
	"homeessentials/backend/internal/mail"
	"homeessentials/backend/internal/middleware"
	"homeessentials/backend/internal/paystack"
	"homeessentials/backend/internal/repository"
	"homeessentials/backend/internal/route"
	"homeessentials/backend/internal/storage"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}

	log, logCloser, err := logging.New(cfg.AppLogFile)
	if err != nil {
		slog.Error("logger setup failed", "err", err)
		os.Exit(1)
	}
	defer logCloser.Close()

	if cfg.AppLogFile != "" {
		log.Info("file logging enabled", "path", cfg.AppLogFile)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	mongo, err := database.Connect(ctx, cfg.MongoURI)
	cancel()
	if err != nil {
		log.Error("mongodb connect failed", "err", err)
		os.Exit(1)
	}

	log.Info("mongodb connected",
		"database", mongo.Database.Name(),
		"cors", middleware.AllowedOriginList(cfg.CORSOrigins),
	)

	indexCtx, indexCancel := context.WithTimeout(context.Background(), 15*time.Second)
	if err := repository.NewProductRepository(mongo.Database).EnsureIndexes(indexCtx); err != nil {
		indexCancel()
		log.Error("product indexes failed", "err", err)
		os.Exit(1)
	}
	if err := repository.NewAdminRepository(mongo.Database).EnsureIndexes(indexCtx); err != nil {
		indexCancel()
		log.Error("admin indexes failed", "err", err)
		os.Exit(1)
	}
	if err := repository.NewOrderRepository(mongo.Database).EnsureIndexes(indexCtx); err != nil {
		indexCancel()
		log.Error("order indexes failed", "err", err)
		os.Exit(1)
	}
	indexCancel()

	tokens, err := auth.NewTokenIssuer(cfg.JWTSecret, 24*time.Hour)
	if err != nil {
		log.Error("jwt setup failed", "err", err)
		os.Exit(1)
	}

	var storageClient *storage.Client
	var uploadHandler *handler.UploadHandler
	if cfg.SupabaseConfigured() {
		var err error
		storageClient, err = storage.NewClient(storage.Config{
			SupabaseURL:    cfg.SupabaseURL,
			ServiceRoleKey: cfg.SupabaseServiceKey,
			Bucket:         cfg.SupabaseStorageBucket,
		})
		if err != nil {
			log.Error("supabase storage setup failed", "err", err)
			os.Exit(1)
		}
		uploadHandler = handler.NewUploadHandler(controller.NewUploadController(storageClient), log)
		log.Info("supabase storage enabled", "bucket", cfg.SupabaseStorageBucket)
	} else {
		log.Info("supabase storage not configured; POST /admin/uploads will return 503")
		uploadHandler = handler.NewUploadHandler(controller.NewUploadController(nil), log)
	}

	if uploadHandler == nil {
		log.Error("upload handler is required")
		os.Exit(1)
	}

	orderRepo := repository.NewOrderRepository(mongo.Database)

	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	defer cleanupCancel()
	go jobs.RunAbandonedOrderCleanup(cleanupCtx, orderRepo, log)

	var psClient *paystack.Client
	if cfg.PaystackConfigured() {
		psClient = paystack.NewClient(cfg.PaystackSecretKey, cfg.PaystackCallbackURL)
		log.Info("paystack enabled", "callbackURL", cfg.PaystackCallbackURL)
	} else {
		log.Info("paystack not configured; payments and webhooks will return 503")
	}

	var mailer *mail.Sender
	if cfg.SMTPConfigured() {
		mailer = mail.NewSender(mail.Config{
			Host:     cfg.SMTPHost,
			Port:     cfg.SMTPPort,
			User:     cfg.SMTPUser,
			Password: cfg.SMTPPassword,
			From:     cfg.SMTPFrom,
		})
		log.Info("smtp enabled", "from", cfg.SMTPFrom)
	} else {
		log.Info("smtp not configured; payment emails will be skipped")
	}

	paymentCtrl := controller.NewPaymentController(orderRepo, psClient, mailer, cfg.AdminNotifyEmail, cfg.ClientPublicURL, log)
	paymentHandler := handler.NewPaymentHandler(paymentCtrl, cfg.PaystackSecretKey)

	productRepo := repository.NewProductRepository(mongo.Database)
	orderCtrl := controller.NewOrderController(orderRepo, productRepo, mailer, log, cfg.ClientPublicURL)
	orderHandler := handler.NewOrderHandler(orderCtrl)

	router := route.NewRouter(cfg, mongo, tokens, uploadHandler, orderHandler, paymentHandler, orderRepo, storageClient, log)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Info("http server listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down")

	cleanupCancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown failed", "err", err)
	}

	if err := mongo.Close(shutdownCtx); err != nil {
		log.Error("mongodb disconnect failed", "err", err)
	}

	log.Info("shutdown complete")
}

