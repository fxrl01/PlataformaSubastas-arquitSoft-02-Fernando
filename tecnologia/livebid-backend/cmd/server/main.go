package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fxrl01/livebid-backend/internal/adapters/handlers"
	"github.com/fxrl01/livebid-backend/internal/adapters/persistence"
	"github.com/fxrl01/livebid-backend/internal/adapters/security"
	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/fxrl01/livebid-backend/internal/core/services"
	"github.com/jackc/pgx/v5/pgxpool"
)

// InMemorySubastaRepo es una implementación ligera en memoria para desarrollo local.
type InMemorySubastaRepo struct {
	subastas map[string]*domain.Subasta
}

func (r *InMemorySubastaRepo) ObtenerPorID(_ context.Context, id string) (*domain.Subasta, error) {
	s, ok := r.subastas[id]
	if !ok {
		return nil, domain.ErrSubastaNoEncontrada
	}
	return s, nil
}

func (r *InMemorySubastaRepo) Guardar(_ context.Context, s *domain.Subasta) error {
	r.subastas[s.ID] = s
	return nil
}

// LocalLockAdapter implementa BidLockPort localmente.
type LocalLockAdapter struct{}

func (l *LocalLockAdapter) AdquirirLock(_ context.Context, _ string, _ time.Duration) (domain.UnlockFunc, error) {
	return func() error { return nil }, nil
}

// LocalEscrowAdapter implementa PaymentEscrowPort.
type LocalEscrowAdapter struct{}

func (e *LocalEscrowAdapter) ValidarGarantiaActiva(_ context.Context, _, _ string) (bool, error) {
	return true, nil
}
func (e *LocalEscrowAdapter) RetenerGarantia(_ context.Context, _, _ string, _ float64) error {
	return nil
}
func (e *LocalEscrowAdapter) LiberarGarantia(_ context.Context, _, _ string) error {
	return nil
}

// LocalNotifierAdapter implementa RoomNotifierPort.
type LocalNotifierAdapter struct{}

func (n *LocalNotifierAdapter) BroadcastNuevaPuja(_ context.Context, subastaID string, p domain.Puja, extension bool) error {
	log.Printf("[WSS BROADCAST] Sala: %s | Nueva Puja: $%.2f por %s (Anti-Sniping extendido: %t)", subastaID, p.Monto, p.PostorID, extension)
	return nil
}

func (n *LocalNotifierAdapter) BroadcastSubastaCerrada(_ context.Context, subastaID string, ganadora *domain.Puja) error {
	if ganadora != nil {
		log.Printf("[WSS BROADCAST] Sala: %s | ADJUDICADA a %s por $%.2f", subastaID, ganadora.PostorID, ganadora.Monto)
	} else {
		log.Printf("[WSS BROADCAST] Sala: %s | Subasta CERRADA DESIERTA", subastaID)
	}
	return nil
}

func main() {
	log.Println("=================================================================")
	log.Println(" LIVEBID — REALTIME CORE ENGINE (Go 1.23+ Clean Architecture)")
	log.Println(" Optimizado para AWS Free Tier (< 30 MB RAM, Latencia < 5ms)")
	log.Println("=================================================================")

	// 1. Inicializar Repositorio y Datos Semilla
	repo := &InMemorySubastaRepo{
		subastas: make(map[string]*domain.Subasta),
	}

	ahora := time.Now()
	subastaDemo := domain.NuevaSubasta(
		"sub-demo-01",
		"Patek Philippe Grandmaster Chime 6300G",
		"Reloj exclusivo de alta complicacion en oro blanco",
		250000.0,
		ahora,
		ahora.Add(2*time.Hour),
	)
	_ = subastaDemo.Activar()
	_ = repo.Guardar(context.Background(), subastaDemo)

	// 2. Inyección de Dependencias del motor de subastas
	lock := &LocalLockAdapter{}
	escrow := &LocalEscrowAdapter{}
	notifier := &LocalNotifierAdapter{}

	_ = services.NuevoAuctionService(repo, lock, escrow, notifier, nil)

	// 3. PostgreSQL y autenticación
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

	jwtIssuer, err := security.NewJWTIssuer(os.Getenv("JWT_SECRET"))
	if err != nil {
		log.Fatalf("configure jwt: %v", err)
	}
	userRepository := persistence.NewPostgresUserRepository(pool)
	authService := services.NewAuthService(
		userRepository,
		security.NewBcryptHasher(12),
		jwtIssuer,
		persistence.NewPostgresRefreshTokenStore(pool),
	)
	authHandler := handlers.NewAuthHandler(authService)

	// 4. Router HTTP estándar de Go
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"UP","engine":"Go 1.23","architecture":"Clean/Hexagonal"}`))
	})

	mux.HandleFunc("POST /api/v1/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/v1/auth/refresh", authHandler.Refresh)
	mux.Handle("GET /api/v1/auth/me", handlers.RequireAuth(jwtIssuer, userRepository, http.HandlerFunc(authHandler.Me)))
	mux.Handle("PATCH /api/v1/admin/users/{id}/activate",
		handlers.RequireAuth(jwtIssuer, userRepository,
			handlers.RequireRoles(domain.RoleAdmin)(http.HandlerFunc(authHandler.ActivateUser))))

	mux.HandleFunc("GET /api/v1/subastas/demo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		sub, _ := repo.ObtenerPorID(r.Context(), "sub-demo-01")
		resp := fmt.Sprintf(`{"id":"%s","titulo":"%s","precio_actual":%.2f,"estado":"%s"}`,
			sub.ID, sub.Titulo, sub.PrecioActual, sub.Estado)
		_, _ = w.Write([]byte(resp))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// 5. Graceful Shutdown
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

	if err := server.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Error durante shutdown: %v", err)
	}

	log.Println(" Servidor LiveBid cerrado exitosamente.")
}
