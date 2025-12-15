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
	"control_page/internal/adaptor"
	httpDelivery "control_page/internal/delivery/http"
	"control_page/internal/model"
	"control_page/internal/model/enum"
	"control_page/internal/repository"
	"control_page/internal/usecase"
	"control_page/pkg/connection"
)


func Run(cfg *config.Config) error {
	// Initialize MongoDB
	mongoClient, err := connection.NewMongo(cfg.MongoDB.URI, cfg.MongoDB.Database)
	if err != nil {
		return fmt.Errorf("connect mongodb: %w", err)
	}
	defer mongoClient.Close()
	log.Printf("Connected to MongoDB: %s/%s", cfg.MongoDB.URI, cfg.MongoDB.Database)

	// Initialize repositories (all using MongoDB now)
	userRepo := repository.NewUserMongoRepository(mongoClient.Database)
	roleRepo := repository.NewRoleMongoRepository(mongoClient.Database)
	userRoleRepo := repository.NewUserRoleMongoRepository(mongoClient.Database)
	apiKeyRepo := repository.NewAPIKeyMongoRepository(mongoClient.Database)
	switcherRepo := repository.NewSwitcherMongoRepository(mongoClient.Database)
	settingRepo := repository.NewSettingMongoRepository(mongoClient.Database)

	// Create default admin user and roles
	if err := createDefaultAdminMongo(userRepo, roleRepo, userRoleRepo); err != nil {
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
	switcherUseCase := usecase.NewSwitcherUseCase(switcherRepo)
	settingUseCase := usecase.NewSettingUseCase(settingRepo)

	// Initialize router
	router := httpDelivery.NewRouter(authUseCase, klineUseCase, roleUseCase, userUseCase, apiKeyUseCase, apiKeyRepo, switcherUseCase, settingUseCase, cfg.Binance.WebSocketURL)

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

	/* Graceful shutdown */
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case <-sigterm:
		log.Println("Shutting down server...")
	}

	// Close WebSocket connections first (this will stop goroutines)
	log.Println("Closing WebSocket connections...")
	router.Close()

	// Graceful shutdown HTTP server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use a channel to track shutdown completion
	shutdownDone := make(chan struct{})
	go func() {
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("server shutdown error: %v", err)
		}
		close(shutdownDone)
	}()

	// Wait for shutdown or force exit after timeout
	select {
	case <-shutdownDone:
		log.Println("Server stopped gracefully")
	case <-ctx.Done():
		log.Println("Server shutdown timed out, forcing exit")
	}

	return nil
}

func createDefaultAdminMongo(
	userRepo adaptor.UserRepository,
	roleRepo adaptor.RoleRepository,
	userRoleRepo adaptor.UserRoleRepository,
) error {
	ctx := context.Background()

	// Check if admin role exists, create if not
	adminRole, err := roleRepo.GetByName(ctx, "admin")
	if err != nil {
		return fmt.Errorf("check admin role: %w", err)
	}
	if adminRole == nil {
		// Create admin role
		adminRole = &model.Role{
			Name:        "admin",
			Description: "Administrator with full access",
		}
		if err := roleRepo.Create(ctx, adminRole); err != nil {
			return fmt.Errorf("create admin role: %w", err)
		}
		log.Println("Created admin role")

		// Add all permissions to admin role
		allPermissions := enum.AllPermissions()
		if err := roleRepo.SetPermissions(ctx, adminRole.ID, allPermissions); err != nil {
			return fmt.Errorf("set admin permissions: %w", err)
		}
		log.Printf("Added %d permissions to admin role", len(allPermissions))
	}

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

	// Assign admin role to user
	if err := userRoleRepo.AssignRole(ctx, adminUser.ID, adminRole.ID); err != nil {
		return fmt.Errorf("assign admin role: %w", err)
	}

	log.Printf("Default admin user created (username: admin, password: admin)")
	return nil
}

