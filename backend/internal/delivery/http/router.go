package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"control_page/internal/adaptor"
	"control_page/internal/model/enum"
)

type Router struct {
	authHandler          *AuthHandler
	klineHandler         *KlineHandler
	rbacHandler          *RBACHandler
	apiKeyHandler        *APIKeyHandler
	switcherHandler      *SwitcherHandler
	settingHandler       *SettingHandler
	btccProxyHandler     *BTCCProxyHandler
	wsManager            *BinanceStreamManager
	tradingStreamManager *TradingStreamManager
	authMiddleware       *AuthMiddleware
}

func NewRouter(
	authUseCase adaptor.AuthUseCase,
	klineUseCase adaptor.KlineUseCase,
	roleUseCase adaptor.RoleUseCase,
	userUseCase adaptor.UserUseCase,
	apiKeyUseCase adaptor.APIKeyUseCase,
	apiKeyRepo adaptor.APIKeyRepository,
	switcherUseCase adaptor.SwitcherUseCase,
	settingUseCase adaptor.SettingUseCase,
	binanceURL string,
) *Router {
	return &Router{
		authHandler:          NewAuthHandler(authUseCase),
		klineHandler:         NewKlineHandler(klineUseCase),
		rbacHandler:          NewRBACHandler(roleUseCase, userUseCase),
		apiKeyHandler:        NewAPIKeyHandler(apiKeyUseCase),
		switcherHandler:      NewSwitcherHandler(switcherUseCase),
		settingHandler:       NewSettingHandler(settingUseCase),
		btccProxyHandler:     NewBTCCProxyHandler(),
		wsManager:            NewBinanceStreamManager(binanceURL),
		tradingStreamManager: NewTradingStreamManager(apiKeyUseCase, authUseCase, apiKeyRepo),
		authMiddleware:       NewAuthMiddleware(authUseCase),
	}
}

func (rt *Router) Setup() *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:5173", "http://localhost:8888"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// API routes
	r.Route("/api", func(r chi.Router) {
		// Auth routes (public + protected)
		r.Route("/auth", func(r chi.Router) {
			// Public
			r.Post("/login", rt.authHandler.Login)
			r.Post("/verify-totp", rt.authHandler.VerifyTOTP)

			// Protected
			r.Group(func(r chi.Router) {
				r.Use(rt.authMiddleware.Authenticate)

				r.Get("/me", rt.authHandler.Me)
				r.Post("/change-password", rt.authHandler.ChangePassword)

				// Registration flows (only admins with manage:users)
				r.Group(func(r chi.Router) {
					r.Use(rt.authMiddleware.RequirePermission(enum.PermissionManageUsers))
					r.Post("/register", rt.authHandler.Register)
					r.Post("/activate", rt.authHandler.ActivateAccount)
				})

				// 2FA rebind routes
				r.Post("/totp/rebind", rt.authHandler.SetupTOTPRebind)
				r.Post("/totp/rebind/confirm", rt.authHandler.ConfirmTOTPRebind)
				r.Post("/totp/rebind/cancel", rt.authHandler.CancelTOTPRebind)
			})
		})

		// Protected routes (requires authentication)
		r.Group(func(r chi.Router) {
			r.Use(rt.authMiddleware.Authenticate)

			// Kline routes (require view:kline permission)
			r.Route("/kline", func(r chi.Router) {
				r.Use(rt.authMiddleware.RequirePermission(enum.PermissionViewKline))
				r.Get("/symbols", rt.klineHandler.GetSymbols)
				r.Get("/intervals", rt.klineHandler.GetIntervals)
			})

			// BTCC proxy routes (require view:kline permission)
			r.Route("/btcc", func(r chi.Router) {
				r.Use(rt.authMiddleware.RequirePermission(enum.PermissionViewKline))
				r.Get("/markets", rt.btccProxyHandler.GetMarketList)
			})

			// RBAC routes
			r.Route("/rbac", func(r chi.Router) {
				// Roles (require manage:roles)
				r.Group(func(r chi.Router) {
					r.Use(rt.authMiddleware.RequirePermission(enum.PermissionManageRoles))
					r.Get("/roles", rt.rbacHandler.ListRoles)
					r.Post("/roles", rt.rbacHandler.CreateRole)
					r.Get("/roles/{id}", rt.rbacHandler.GetRole)
					r.Put("/roles/{id}", rt.rbacHandler.UpdateRole)
					r.Delete("/roles/{id}", rt.rbacHandler.DeleteRole)
					r.Put("/roles/{id}/permissions", rt.rbacHandler.SetRolePermissions)
					r.Get("/permissions", rt.rbacHandler.GetAllPermissions)
				})

				// Users (require manage:users)
				r.Group(func(r chi.Router) {
					r.Use(rt.authMiddleware.RequirePermission(enum.PermissionManageUsers))
					r.Get("/users", rt.rbacHandler.ListUsers)
					r.Post("/users", rt.rbacHandler.CreateUser)
					r.Get("/users/{id}", rt.rbacHandler.GetUser)
					r.Put("/users/{id}", rt.rbacHandler.UpdateUser)
					r.Delete("/users/{id}", rt.rbacHandler.DeleteUser)
					r.Post("/users/{id}/roles", rt.rbacHandler.AssignRole)
					r.Delete("/users/{id}/roles/{roleId}", rt.rbacHandler.RemoveRole)
					r.Post("/users/{id}/totp/reset", rt.rbacHandler.ResetUserTOTP)
				})
			})

			// API Keys routes
			r.Route("/api-keys", func(r chi.Router) {
				// View routes (require view:api_keys permission)
				r.Group(func(r chi.Router) {
					r.Use(rt.authMiddleware.RequirePermission(enum.PermissionViewAPIKeys))
					r.Get("/", rt.apiKeyHandler.List)
					r.Get("/platforms", rt.apiKeyHandler.GetPlatforms)
					r.Get("/{id}", rt.apiKeyHandler.Get)
				})
				// Manage routes (require manage:api_keys permission)
				r.Group(func(r chi.Router) {
					r.Use(rt.authMiddleware.RequirePermission(enum.PermissionManageAPIKeys))
					r.Post("/", rt.apiKeyHandler.Create)
					r.Put("/{id}", rt.apiKeyHandler.Update)
					r.Delete("/{id}", rt.apiKeyHandler.Delete)
				})
			})

			// Switcher routes
			r.Route("/switchers", func(r chi.Router) {
				// View routes (require view:settings permission)
				r.Group(func(r chi.Router) {
					r.Use(rt.authMiddleware.RequirePermission(enum.PermissionViewSettings))
					r.Get("/", rt.switcherHandler.List)
					r.Get("/{id}", rt.switcherHandler.Get)
				})
				// Manage routes (require manage:settings permission)
				r.Group(func(r chi.Router) {
					r.Use(rt.authMiddleware.RequirePermission(enum.PermissionManageSettings))
					r.Post("/", rt.switcherHandler.Create)
					r.Put("/{id}", rt.switcherHandler.Update)
					r.Put("/{id}/pairs/{pair}", rt.switcherHandler.UpdatePair)
					r.Delete("/{id}", rt.switcherHandler.Delete)
				})
			})

			// Setting routes
			r.Route("/settings", func(r chi.Router) {
				// View routes (require view:settings permission)
				r.Group(func(r chi.Router) {
					r.Use(rt.authMiddleware.RequirePermission(enum.PermissionViewSettings))
					r.Get("/", rt.settingHandler.List)
					r.Get("/search", rt.settingHandler.GetByBaseQuote)
					r.Get("/{id}", rt.settingHandler.Get)
				})
				// Manage routes (require manage:settings permission)
				r.Group(func(r chi.Router) {
					r.Use(rt.authMiddleware.RequirePermission(enum.PermissionManageSettings))
					r.Post("/", rt.settingHandler.Create)
					r.Put("/{id}", rt.settingHandler.Update)
					r.Put("/{id}/parameters/{strategy}", rt.settingHandler.UpdateParameters)
					r.Delete("/{id}", rt.settingHandler.Delete)
				})
			})
		})
	})

	// WebSocket routes (handled separately, auth via query param)
	r.Get("/ws/kline", rt.wsManager.HandleWebSocket)
	r.Get("/ws/trading", rt.tradingStreamManager.HandleWebSocket)

	return r
}

func (rt *Router) Close() {
	rt.wsManager.Close()
	rt.tradingStreamManager.Close()
}
