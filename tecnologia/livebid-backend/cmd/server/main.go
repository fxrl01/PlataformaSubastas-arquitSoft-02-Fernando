package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/fxrl01/livebid-backend/internal/adapters/ai"
	"github.com/fxrl01/livebid-backend/internal/adapters/handlers"
	"github.com/fxrl01/livebid-backend/internal/adapters/payments"
	"github.com/fxrl01/livebid-backend/internal/adapters/persistence"
	"github.com/fxrl01/livebid-backend/internal/adapters/realtime"
	"github.com/fxrl01/livebid-backend/internal/adapters/security"
	"github.com/fxrl01/livebid-backend/internal/adapters/web3"
	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/fxrl01/livebid-backend/internal/core/services"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func main() {
	log.Println("=================================================================")
	log.Println(" LIVEBID — REALTIME CORE ENGINE (Go 1.23+ Clean Architecture)")
	log.Println(" Optimizado para AWS Free Tier (< 30 MB RAM, Latencia < 5ms)")
	log.Println("=================================================================")

	// 1. Conexión a PostgreSQL y migración idempotente de esquema
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
	pool, err := pgxpool.New(dbCtx, databaseURL)
	dbCancel()
	if err != nil {
		log.Fatalf("open database pool: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("ping database: %v", err)
	}
	if err := persistence.EnsureSchema(context.Background(), pool); err != nil {
		log.Printf("advertencia en migración automática de esquema: %v", err)
	}

	// 2. Conexión a Redis (Locks Atómicos + Pub/Sub)
	redisClient := redis.NewClient(&redis.Options{Addr: redisAddr(os.Getenv("REDIS_URL"))})
	defer redisClient.Close()
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("ping redis: %v", err)
	}

	// 3. Seguridad, Repositorios y Caché TTL
	jwtIssuer, err := security.NewJWTIssuer(os.Getenv("JWT_SECRET"))
	if err != nil {
		log.Fatalf("configure jwt: %v", err)
	}
	userRepository := persistence.NewCachedUserRepository(
		persistence.NewPostgresUserRepository(pool),
		30*time.Second,
	)
	auctionRepository := persistence.NewPostgresAuctionRepository(pool)
	bidRepository := persistence.NewPostgresBidRepository(pool)
	guaranteeRepository := persistence.NewPostgresGuaranteeRepository(pool)
	bidLock := persistence.NewRedisBidLock(redisClient)
	bidNotifier := realtime.NewRedisBidNotifier(redisClient)

	// 4. Gateway Multi-Modelo de IA + Circuit Breaker + Detector Asíncrono + Capa Web3
	failThreshold := envInt("CIRCUIT_BREAKER_FAIL_THRESHOLD", 3)
	cbTimeout := time.Duration(envInt("CIRCUIT_BREAKER_TIMEOUT_MS", 2000)) * time.Millisecond
	geminiProvider := ai.NewGeminiRiskAdapter(os.Getenv("GEMINI_API_KEY"))
	riskBreaker := ai.NewCircuitBreakerIA(geminiProvider, failThreshold, 3*time.Second, cbTimeout)

	aiRouter := ai.NewMultiModelRouter("gemini-2.5-flash", riskBreaker, time.Minute)
	aiRouter.RegisterModel("gemini-2.5-pro-forensic", riskBreaker)
	fraudDetector := ai.NewAsyncFraudDetector(aiRouter)

	chainID := int64(envInt("WEB3_CHAIN_ID", 8453))
	escrowContract := os.Getenv("WEB3_ESCROW_CONTRACT")
	eip712Verifier := web3.NewEIP712Verifier(chainID, escrowContract)
	blockchainEscrow := web3.NewEVMEscrowAdapter(chainID, escrowContract, os.Getenv("WEB3_RPC_URL"))
	stripeGateway := payments.NewStripeAdapter(
		os.Getenv("STRIPE_SECRET_KEY"),
		os.Getenv("STRIPE_PUBLISHABLE_KEY"),
		os.Getenv("STRIPE_WEBHOOK_SECRET"),
	)
	guaranteeRepository.WithPaymentGateway(stripeGateway)

	// 5. Servicios de Aplicación y Bootstrap Automático de Administrador
	authService := services.NewAuthService(
		userRepository,
		security.NewBcryptHasher(12),
		jwtIssuer,
		persistence.NewPostgresRefreshTokenStore(pool),
	)
	adminEmail := envStr("ADMIN_BOOTSTRAP_EMAIL", "admin@livebid.pe")
	adminPass := envStr("ADMIN_BOOTSTRAP_PASSWORD", "AdminLiveBid2026!")
	if adminView, err := authService.BootstrapAdmin(context.Background(), "Administrador General LiveBid", adminEmail, adminPass); err != nil {
		log.Printf("advertencia en bootstrap de admin: %v", err)
	} else {
		log.Printf("Cuenta ADMIN verificada y activa: %s (ID: %s)", adminView.Email, adminView.ID)
	}

	catalogService := services.NuevoAuctionCatalogServiceConGarantiasYNotificador(
		auctionRepository,
		guaranteeRepository,
		bidNotifier,
	).WithWeb3Settlement(bidRepository, blockchainEscrow)
	bidService := services.NewAuctionBidServiceWithNotifierAndGuarantee(
		auctionRepository,
		bidRepository,
		bidLock,
		bidNotifier,
		guaranteeRepository,
	).WithWeb3Verifier(eip712Verifier).WithFraudTelemetry(fraudDetector)
	guaranteeService := services.NewGuaranteeService(auctionRepository, guaranteeRepository, aiRouter).
		WithBlockchainEscrow(blockchainEscrow).
		WithPaymentGateway(stripeGateway)
	bidHistoryService := services.NewBidHistoryService(auctionRepository, bidRepository)
	web3AuditService := services.NewWeb3AuditService(auctionRepository, bidRepository)
	agentDelegationService := services.NewAgentDelegationService(auctionRepository, bidService)

	authHandler := handlers.NewAuthHandler(authService)
	auctionHandler := handlers.NewAuctionHandler(catalogService, web3AuditService)
	bidHandler := handlers.NewBidHandler(bidService)
	bidHistoryHandler := handlers.NewBidHistoryHandler(bidHistoryService)
	guaranteeHandler := handlers.NewGuaranteeHandler(guaranteeService)
	agentAIHandler := handlers.NewAgentAIHandler(agentDelegationService, aiRouter, riskBreaker, fraudDetector)
	webSocketHandler := realtime.NewWebSocketHandler(jwtIssuer, userRepository, auctionRepository, redisClient)

	// 6. Router HTTP estándar de Go
	mux := http.NewServeMux()

	publicDir := resolvePublicDir()
	if publicDir != "" {
		fileServer := http.FileServer(http.Dir(publicDir))
		mux.Handle("GET /{$}", fileServer)
		mux.Handle("GET /index.html", fileServer)
		mux.Handle("GET /styles.css", fileServer)
		mux.Handle("GET /app.js", fileServer)
	}

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := fmt.Sprintf(`{"status":"UP","engine":"Go 1.23","architecture":"Clean/Hexagonal","circuit_breaker_ia":"%s","active_ai_model":"%s","web3_chain_id":%d}`,
			riskBreaker.State(), aiRouter.ActiveModel(), chainID)
		_, _ = w.Write([]byte(resp))
	})

	// Autenticación y Perfil
	mux.HandleFunc("POST /api/v1/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/v1/auth/refresh", authHandler.Refresh)
	mux.Handle("GET /api/v1/auth/me", handlers.RequireAuth(jwtIssuer, userRepository, http.HandlerFunc(authHandler.Me)))

	// Gobernanza RBAC y Administración de Usuarios (Rol ADMIN)
	mux.Handle("GET /api/v1/admin/users",
		handlers.RequireAuth(jwtIssuer, userRepository,
			handlers.RequireRoles(domain.RoleAdmin)(http.HandlerFunc(authHandler.ListUsers))))
	mux.Handle("PATCH /api/v1/admin/users/{id}/activate",
		handlers.RequireAuth(jwtIssuer, userRepository,
			handlers.RequireRoles(domain.RoleAdmin)(http.HandlerFunc(authHandler.ActivateUser))))
	mux.Handle("PATCH /api/v1/admin/users/{id}/role",
		handlers.RequireAuth(jwtIssuer, userRepository,
			handlers.RequireRoles(domain.RoleAdmin)(http.HandlerFunc(authHandler.UpdateUserRole))))
	mux.Handle("PATCH /api/v1/admin/users/{id}/status",
		handlers.RequireAuth(jwtIssuer, userRepository,
			handlers.RequireRoles(domain.RoleAdmin)(http.HandlerFunc(authHandler.UpdateUserStatus))))
	mux.Handle("PATCH /api/v1/admin/ia/modelo",
		handlers.RequireAuth(jwtIssuer, userRepository,
			handlers.RequireRoles(domain.RoleAdmin, domain.RoleAuctioneer)(http.HandlerFunc(agentAIHandler.SwitchModel))))

	// Catálogo de Subastas, Encargado Oficial, Ficha del Lote y Ciclo de Vida
	mux.HandleFunc("GET /api/v1/subastas", auctionHandler.List)
	mux.HandleFunc("GET /api/v1/subastas/{id}", auctionHandler.Get)
	mux.HandleFunc("GET /api/v1/subastas/{id}/pujas", bidHistoryHandler.List)
	mux.HandleFunc("GET /api/v1/subastas/{id}/auditoria-web3", auctionHandler.AuditWeb3)
	mux.HandleFunc("GET /api/v1/subastas/{id}/inteligencia-ia", agentAIHandler.IntelligenceReport)
	mux.Handle("POST /api/v1/subastas",
		handlers.RequireAuth(jwtIssuer, userRepository,
			handlers.RequireRoles(domain.RoleSeller, domain.RoleAuctioneer, domain.RoleAdmin)(http.HandlerFunc(auctionHandler.Create))))
	mux.Handle("PUT /api/v1/subastas/{id}/encargado",
		handlers.RequireAuth(jwtIssuer, userRepository,
			handlers.RequireRoles(domain.RoleSeller, domain.RoleAuctioneer, domain.RoleAdmin)(http.HandlerFunc(auctionHandler.UpdateEncargado))))
	mux.Handle("PUT /api/v1/subastas/{id}/detalles-lote",
		handlers.RequireAuth(jwtIssuer, userRepository,
			handlers.RequireRoles(domain.RoleSeller, domain.RoleAuctioneer, domain.RoleAdmin)(http.HandlerFunc(auctionHandler.UpdateDetallesLote))))
	mux.Handle("PATCH /api/v1/subastas/{id}/activate",
		handlers.RequireAuth(jwtIssuer, userRepository,
			handlers.RequireRoles(domain.RoleAuctioneer, domain.RoleAdmin)(http.HandlerFunc(auctionHandler.Activate))))
	mux.Handle("POST /api/v1/subastas/{id}/activar",
		handlers.RequireAuth(jwtIssuer, userRepository,
			handlers.RequireRoles(domain.RoleAuctioneer, domain.RoleAdmin)(http.HandlerFunc(auctionHandler.Activate))))
	mux.Handle("PATCH /api/v1/subastas/{id}/countdown",
		handlers.RequireAuth(jwtIssuer, userRepository,
			handlers.RequireRoles(domain.RoleAuctioneer, domain.RoleAdmin)(http.HandlerFunc(auctionHandler.StartCountdown))))
	mux.Handle("POST /api/v1/subastas/{id}/conteo",
		handlers.RequireAuth(jwtIssuer, userRepository,
			handlers.RequireRoles(domain.RoleAuctioneer, domain.RoleAdmin)(http.HandlerFunc(auctionHandler.StartCountdown))))
	mux.Handle("PATCH /api/v1/subastas/{id}/close",
		handlers.RequireAuth(jwtIssuer, userRepository,
			handlers.RequireRoles(domain.RoleAuctioneer, domain.RoleAdmin)(http.HandlerFunc(auctionHandler.Close))))
	mux.Handle("POST /api/v1/subastas/{id}/cerrar",
		handlers.RequireAuth(jwtIssuer, userRepository,
			handlers.RequireRoles(domain.RoleAuctioneer, domain.RoleAdmin)(http.HandlerFunc(auctionHandler.Close))))

	// Pujas, Garantías (Stripe PaymentIntents Hold) y Agentes Autónomos de IA
	mux.Handle("POST /api/v1/subastas/{id}/pujas",
		handlers.RequireAuth(jwtIssuer, userRepository,
			handlers.RequireRoles(domain.RoleBidder, domain.RoleAdmin)(http.HandlerFunc(bidHandler.Place))))
	mux.Handle("GET /api/v1/subastas/{id}/garantias/me",
		handlers.RequireAuth(jwtIssuer, userRepository,
			http.HandlerFunc(guaranteeHandler.GetMyGuarantee)))
	mux.Handle("POST /api/v1/subastas/{id}/garantias/stripe-intent",
		handlers.RequireAuth(jwtIssuer, userRepository,
			handlers.RequireRoles(domain.RoleBidder, domain.RoleAdmin)(http.HandlerFunc(guaranteeHandler.PrepareStripeIntent))))
	mux.Handle("POST /api/v1/subastas/{id}/garantias",
		handlers.RequireAuth(jwtIssuer, userRepository,
			handlers.RequireRoles(domain.RoleBidder, domain.RoleAdmin)(http.HandlerFunc(guaranteeHandler.Create))))
	mux.HandleFunc("POST /api/v1/webhooks/stripe", guaranteeHandler.StripeWebhook)
	mux.Handle("POST /api/v1/subastas/{id}/agentes-ia/delegar",
		handlers.RequireAuth(jwtIssuer, userRepository,
			handlers.RequireRoles(domain.RoleBidder, domain.RoleAdmin)(http.HandlerFunc(agentAIHandler.Delegate))))
	mux.Handle("POST /api/v1/subastas/{id}/agentes-ia/auto-puja",
		handlers.RequireAuth(jwtIssuer, userRepository,
			handlers.RequireRoles(domain.RoleBidder, domain.RoleAdmin)(http.HandlerFunc(agentAIHandler.AutoBid))))
	mux.Handle("GET /ws/subastas/{id}", webSocketHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      withCORS(mux),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// 7. Reconciliación inicial y Scheduler de Cierre Automático
	schedulerContext, cancelScheduler := context.WithCancel(context.Background())
	defer cancelScheduler()
	if closed, err := catalogService.CerrarVencidas(schedulerContext, time.Now()); err != nil {
		log.Printf("advertencia al reconciliar subastas vencidas al inicio: %v", err)
	} else if closed > 0 {
		log.Printf("Reconciliación inicial: %d subasta(s) vencida(s) cerrada(s) y liquidada(s)", closed)
	}
	go services.NewAuctionScheduler(catalogService, 10*time.Second).Run(schedulerContext)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf(" Servidor HTTP escuchando en http://localhost:%s\n", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error en servidor: %v", err)
		}
	}()

	<-stop
	log.Println("Señal de apagado recibida. Cerrando conexiones limpiamente...")

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cancelScheduler()

	if err := server.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Error durante shutdown: %v", err)
	}

	log.Println(" Servidor LiveBid cerrado exitosamente.")
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Idempotency-Key, X-Web3-Wallet, X-Web3-Signature, X-Web3-Nonce, X-Web3-Deadline, X-AI-Agent-Mandate")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func redisAddr(rawURL string) string {
	if rawURL == "" {
		return "localhost:6379"
	}
	const prefix = "redis://"
	if len(rawURL) >= len(prefix) && rawURL[:len(prefix)] == prefix {
		return rawURL[len(prefix):]
	}
	return rawURL
}

func envInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(val)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func envStr(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}

func resolvePublicDir() string {
	candidates := []string{
		"/app/public",
		"public",
		filepath.Join("..", "boilerplate-clean-arquitecture", "public"),
	}
	for _, dir := range candidates {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	return ""
}
