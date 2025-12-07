package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/crypto/bcrypt"

	"control_page/config"
	"control_page/database"
	httpDelivery "control_page/internal/delivery/http"
	"control_page/internal/model"
	"control_page/internal/repository"
	"control_page/internal/usecase"
	"control_page/pkg/connection"
)

func Run(cfg *config.Config) error {
	// Initialize database
	db, err := connection.NewSQLite(cfg.Database.DSN)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.RunMigrations(db); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	userRoleRepo := repository.NewUserRoleRepository(db)
	apiKeyRepo := repository.NewAPIKeyRepository(db)

	// Create default admin user
	if err := createDefaultAdmin(userRepo, roleRepo, userRoleRepo); err != nil {
		log.Printf("Warning: failed to create default admin: %v", err)
	}

	// Initialize use cases
	authUseCase := usecase.NewAuthUseCase(
		userRepo,
		roleRepo,
		userRoleRepo,
		cfg.JWT.Secret,
		cfg.JWT.Expiration,
	)
	klineUseCase := usecase.NewKlineUseCase()
	roleUseCase := usecase.NewRoleUseCase(roleRepo)
	userUseCase := usecase.NewUserUseCase(userRepo, roleRepo, userRoleRepo)
	apiKeyUseCase := usecase.NewAPIKeyUseCase(apiKeyRepo)

	// Initialize router
	router := httpDelivery.NewRouter(authUseCase, klineUseCase, roleUseCase, userUseCase, apiKeyUseCase, apiKeyRepo, cfg.Binance.WebSocketURL)
	defer router.Close()

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      router.Setup(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		log.Printf("Server starting on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case <-quit:
		log.Println("Shutting down server...")
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	log.Println("Server stopped")
	return nil
}

func createDefaultAdmin(
	userRepo *repository.UserRepository,
	roleRepo *repository.RoleRepository,
	userRoleRepo *repository.UserRoleRepository,
) error {
	ctx := context.Background()

	// Check if admin user already exists
	existingUser, err := userRepo.GetByUsername(ctx, "admin")
	if err != nil {
		return fmt.Errorf("check existing admin: %w", err)
	}
	if existingUser != nil {
		log.Println("Default admin user already exists")
		return nil
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	// Create admin user
	adminUser := &model.User{
		Username: "admin",
		Email:    "admin@example.com",
		Password: string(hashedPassword),
		IsActive: true,
	}

	if err := userRepo.Create(ctx, adminUser); err != nil {
		return fmt.Errorf("create admin user: %w", err)
	}

	// Get admin role
	adminRole, err := roleRepo.GetByName(ctx, "admin")
	if err != nil {
		return fmt.Errorf("get admin role: %w", err)
	}
	if adminRole == nil {
		return fmt.Errorf("admin role not found")
	}

	// Assign admin role to user
	if err := userRoleRepo.AssignRole(ctx, adminUser.ID, adminRole.ID); err != nil {
		return fmt.Errorf("assign admin role: %w", err)
	}

	log.Printf("Default admin user created (username: admin, password: admin)")
	return nil
}
