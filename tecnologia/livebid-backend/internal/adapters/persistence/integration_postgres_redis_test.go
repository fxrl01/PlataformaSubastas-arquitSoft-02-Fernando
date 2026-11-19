package persistence_test

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fxrl01/livebid-backend/internal/adapters/payments"
	"github.com/fxrl01/livebid-backend/internal/adapters/persistence"
	"github.com/fxrl01/livebid-backend/internal/adapters/realtime"
	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/redis/go-redis/v9"
)

func loadEnvFallback(key string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for i := 0; i < 7; i++ {
		p := filepath.Join(dir, ".env")
		if f, err := os.Open(p); err == nil {
			sc := bufio.NewScanner(f)
			for sc.Scan() {
				line := strings.TrimSpace(sc.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				kv := strings.SplitN(line, "=", 2)
				if len(kv) == 2 && strings.TrimSpace(kv[0]) == key {
					_ = f.Close()
					return strings.TrimSpace(kv[1])
				}
			}
			_ = f.Close()
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func TestIntegrationPostgresAndRedisEndToEndRepositories(t *testing.T) {
	dbURL := loadEnvFallback("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = loadEnvFallback("DATABASE_URL")
	}
	if dbURL == "" {
		if pw := loadEnvFallback("POSTGRES_PASSWORD"); pw != "" {
			usr := loadEnvFallback("POSTGRES_USER")
			if usr == "" {
				usr = "livebid_admin"
			}
			dbURL = "postgres://" + usr + ":" + pw + "@localhost:5432/livebid_db"
		}
	}
	redisURL := loadEnvFallback("TEST_REDIS_URL")
	if redisURL == "" {
		redisURL = loadEnvFallback("REDIS_URL")
	}
	if redisURL == "" {
		redisURL = "localhost:6379"
	}
	redisAddr := strings.TrimPrefix(redisURL, "redis://")

	if dbURL == "" {
		t.Skip("DATABASE_URL/TEST_DATABASE_URL not configured; skipping live DB integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := persistence.NewPostgresPool(ctx, dbURL, persistence.DefaultPoolOptionsFromEnv())
	if err != nil {
		t.Skipf("postgres unreachable (%v); skipping live DB integration test", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("postgres ping failed (%v); skipping live DB integration test", err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer rdb.Close()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis unreachable (%v); skipping live Redis integration test", err)
	}

	if err := persistence.EnsureSchema(ctx, pool); err != nil {
		t.Fatalf("EnsureSchema failed: %v", err)
	}

	suffix := time.Now().Format("20060102150405.000000")
	suffix = strings.ReplaceAll(suffix, ".", "")
	sellerID := "it_seller_" + suffix
	bidderID := "it_bidder_" + suffix
	loserID := "it_loser_" + suffix
	auctionID := "it_sub_" + suffix

	defer func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanCancel()
		_, _ = pool.Exec(cleanCtx, `DELETE FROM logs_auditoria WHERE subasta_id = $1`, auctionID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM subastas WHERE id = $1`, auctionID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM refresh_tokens WHERE user_id IN ($1, $2, $3)`, sellerID, bidderID, loserID)
		_, _ = pool.Exec(cleanCtx, `DELETE FROM usuarios WHERE id IN ($1, $2, $3)`, sellerID, bidderID, loserID)
	}()

	// 1. UserRepository + CachedUserRepository con invalidación Pub/Sub Redis + RefreshTokenStore
	pgUsers := persistence.NewPostgresUserRepository(pool)
	cacheCtx, cancelCache := context.WithCancel(ctx)
	defer cancelCache()
	cachedUsersA := persistence.NewCachedUserRepository(pgUsers, 30*time.Second).WithRedisInvalidation(cacheCtx, rdb)
	cachedUsersB := persistence.NewCachedUserRepository(pgUsers, 30*time.Second).WithRedisInvalidation(cacheCtx, rdb)

	now := time.Now().UTC()
	seller, _ := domain.NewUser(sellerID, "Seller IT", sellerID+"@livebid.pe", "bcrypt_hash", now)
	bidder, _ := domain.NewUser(bidderID, "Bidder IT", bidderID+"@livebid.pe", "bcrypt_hash", now)
	loser, _ := domain.NewUser(loserID, "Loser IT", loserID+"@livebid.pe", "bcrypt_hash", now)

	for _, u := range []*domain.User{seller, bidder, loser} {
		if err := cachedUsersA.Save(ctx, u); err != nil {
			t.Fatalf("save user %s failed: %v", u.ID, err)
		}
	}
	if _, err := cachedUsersB.FindByID(ctx, seller.ID); err != nil {
		t.Fatalf("cachedUsersB FindByID failed: %v", err)
	}
	if err := cachedUsersA.UpdateRole(ctx, seller.ID, domain.RoleSeller); err != nil {
		t.Fatalf("UpdateRole failed: %v", err)
	}
	if err := cachedUsersA.UpdateKYCAndStatus(ctx, seller.ID, domain.UserActive, true); err != nil {
		t.Fatalf("UpdateKYCAndStatus failed: %v", err)
	}
	if err := cachedUsersA.UpdateStatus(ctx, bidder.ID, domain.UserActive); err != nil {
		t.Fatalf("UpdateStatus bidder failed: %v", err)
	}
	if err := cachedUsersA.UpdateStatus(ctx, loser.ID, domain.UserActive); err != nil {
		t.Fatalf("UpdateStatus loser failed: %v", err)
	}
	if byEmail, err := cachedUsersA.FindByEmail(ctx, seller.Email); err != nil || byEmail.Role != domain.RoleSeller {
		t.Fatalf("FindByEmail failed: %+v err=%v", byEmail, err)
	}
	if list, err := cachedUsersA.ListUsers(ctx, domain.UserActive, domain.RoleSeller); err != nil || len(list) == 0 {
		t.Fatalf("ListUsers failed: len=%d err=%v", len(list), err)
	}

	rtStore := persistence.NewPostgresRefreshTokenStore(pool)
	rtHash := "rthash_" + suffix
	if err := rtStore.Save(ctx, rtHash, bidderID, now.Add(time.Hour)); err != nil {
		t.Fatalf("refresh token Save failed: %v", err)
	}
	if uid, err := rtStore.Consume(ctx, rtHash); err != nil || uid != bidderID {
		t.Fatalf("refresh token Consume failed: uid=%s err=%v", uid, err)
	}
	if _, err := rtStore.Consume(ctx, rtHash); err != domain.ErrRefreshTokenInvalid {
		t.Fatalf("expected ErrRefreshTokenInvalid on second consume, got %v", err)
	}

	// 2. AuctionRepository + CachedAuctionRepository (Redis Cache-Aside real)
	pgAuctions := persistence.NewPostgresAuctionRepositoryWithReadPool(pool, pool)
	cachedAuctions := persistence.NewCachedAuctionRepository(pgAuctions, persistence.NewRedisDistributedCache(rdb), 15*time.Second)

	subasta := domain.NuevaSubasta(auctionID, "Lote Integración Real", "Prueba Postgres+Redis", 1000, now, now.Add(time.Hour))
	subasta.VendedorID = sellerID
	_ = subasta.AsignarEncargado(domain.EncargadoSubasta{Nombre: "Martillero IT", Licencia: "LIC-IT"})
	_ = subasta.ActualizarDetallesLote(domain.DetallesLote{Categoria: "TECH", Ubicacion: "Lima"})
	_ = subasta.Activar()

	if err := cachedAuctions.Guardar(ctx, subasta); err != nil {
		t.Fatalf("Guardar auction failed: %v", err)
	}
	if got, err := cachedAuctions.ObtenerPorID(ctx, auctionID); err != nil || got.Titulo != "Lote Integración Real" {
		t.Fatalf("ObtenerPorID #1 failed: %v", err)
	}
	// Segunda lectura viene desde Redis Cache-Aside
	if got, err := cachedAuctions.ObtenerPorID(ctx, auctionID); err != nil || got.ID != auctionID {
		t.Fatalf("ObtenerPorID #2 (Redis hit) failed: %v", err)
	}
	if list, err := cachedAuctions.Listar(ctx); err != nil || len(list) == 0 {
		t.Fatalf("Listar failed: %v", err)
	}
	if _, err := cachedAuctions.ListarVencidas(ctx, now); err != nil {
		t.Fatalf("ListarVencidas failed: %v", err)
	}

	// 3. RedisBidLock + RedisBidNotifier + AuctionRoomHub
	lockAdapter := persistence.NewRedisBidLock(rdb)
	unlock, err := lockAdapter.AdquirirLock(ctx, auctionID, 1500*time.Millisecond)
	if err != nil {
		t.Fatalf("AdquirirLock failed: %v", err)
	}
	if _, err := lockAdapter.AdquirirLock(ctx, auctionID, 1500*time.Millisecond); err == nil {
		t.Fatal("expected second concurrent lock on same auction to fail")
	}
	if err := unlock(); err != nil {
		t.Fatalf("unlock failed: %v", err)
	}

	hub := realtime.NewAuctionRoomHub(realtime.NewRedisSubscriber(rdb))
	msgCh, leave, err := hub.Join(auctionID)
	if err != nil {
		t.Fatalf("hub.Join failed: %v", err)
	}
	defer leave()

	notifier := realtime.NewRedisBidNotifier(rdb)

	// 4. GuaranteeRepository + StripeAdapter
	stripeGW := payments.NewStripeAdapter("", "pk_test_it", "whsec_it")
	pgGuarantees := persistence.NewPostgresGuaranteeRepository(pool).WithPaymentGateway(stripeGW)
	intentW, _ := stripeGW.CreateManualHoldIntent(ctx, auctionID, bidderID, 100, "usd")
	intentL, _ := stripeGW.CreateManualHoldIntent(ctx, auctionID, loserID, 100, "usd")

	if err := pgGuarantees.Save(ctx, &domain.Guarantee{
		ID: "gar_w_" + suffix, AuctionID: auctionID, BidderID: bidderID,
		Amount: 100, AuthorizationID: intentW.PaymentIntentID, Status: domain.GuaranteeHeld, TrustScore: 90, CreatedAt: now,
	}); err != nil {
		t.Fatalf("save winner guarantee failed: %v", err)
	}
	if err := pgGuarantees.Save(ctx, &domain.Guarantee{
		ID: "gar_l_" + suffix, AuctionID: auctionID, BidderID: loserID,
		Amount: 100, AuthorizationID: intentL.PaymentIntentID, Status: domain.GuaranteeHeld, TrustScore: 80, CreatedAt: now,
	}); err != nil {
		t.Fatalf("save loser guarantee failed: %v", err)
	}
	if active, err := pgGuarantees.FindActive(ctx, auctionID, bidderID); err != nil || active.AuthorizationID != intentW.PaymentIntentID {
		t.Fatalf("FindActive guarantee failed: %v", err)
	}

	// 5. BidRepository (RegistrarPujaAtomica + Idempotency + AuditChain SHA-256 + OnChainReceipt)
	pgBids := persistence.NewPostgresBidRepositoryWithReadPool(pool, pool)
	if err := pgBids.AppendAuditEvent(ctx, auctionID, "SALA_ACTIVADA", map[string]any{"estado": "ACTIVA"}); err != nil {
		t.Fatalf("AppendAuditEvent failed: %v", err)
	}
	bid := domain.NuevaPuja("bid_"+suffix, auctionID, bidderID, 1100, now)
	if err := pgBids.RegistrarPujaAtomica(ctx, auctionID, bid, 1000, now.Add(time.Hour), false, "idem_"+suffix); err != nil {
		t.Fatalf("RegistrarPujaAtomica failed: %v", err)
	}
	if dup, err := pgBids.BuscarPorIdempotencyKey(ctx, auctionID, bidderID, "idem_"+suffix); err != nil || dup == nil || dup.ID != bid.ID {
		t.Fatalf("BuscarPorIdempotencyKey failed: %+v err=%v", dup, err)
	}
	if items, total, err := pgBids.ListarPorSubasta(ctx, auctionID, "", 0, 10, 0); err != nil || total != 1 || len(items) != 1 {
		t.Fatalf("ListarPorSubasta failed: total=%d len=%d err=%v", total, len(items), err)
	}

	// Broadcast en tiempo real vía Redis Pub/Sub -> Hub
	if err := notifier.BroadcastNuevaPuja(ctx, auctionID, bid, false, now.Add(time.Hour)); err != nil {
		t.Fatalf("BroadcastNuevaPuja failed: %v", err)
	}
	select {
	case raw := <-msgCh:
		if !strings.Contains(string(raw), bid.ID) {
			t.Fatalf("expected bid id in pubsub message, got %s", string(raw))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Redis Pub/Sub broadcast on AuctionRoomHub")
	}

	if err := pgBids.GuardarReciboOnChain(ctx, domain.OnChainSettlementReceipt{
		AuctionID: auctionID, MerkleRoot: "0xabc", WinnerID: bidderID, WinningAmount: 1100,
		TxHash: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", ChainID: 8453, AnchoredAt: now,
	}); err != nil {
		t.Fatalf("GuardarReciboOnChain failed: %v", err)
	}
	if rec, err := pgBids.ObtenerReciboOnChain(ctx, auctionID); err != nil || rec == nil || rec.WinnerID != bidderID {
		t.Fatalf("ObtenerReciboOnChain failed: %+v err=%v", rec, err)
	}
	if hashes, err := pgBids.ObtenerHashesAuditoria(ctx, auctionID); err != nil || len(hashes) != 1 {
		t.Fatalf("ObtenerHashesAuditoria failed: len=%d err=%v", len(hashes), err)
	}
	if chain, err := pgBids.ObtenerCadenaAuditoria(ctx, auctionID); err != nil || len(chain) < 3 {
		t.Fatalf("ObtenerCadenaAuditoria failed: len=%d err=%v", len(chain), err)
	}

	// Liquidación real de garantías en Postgres + Stripe Sandbox
	if err := pgGuarantees.SettleAuction(ctx, auctionID, bidderID); err != nil {
		t.Fatalf("SettleAuction failed: %v", err)
	}
	if err := pgGuarantees.UpdateStatusByAuthorizationID(ctx, intentL.PaymentIntentID, domain.GuaranteeReleased); err != nil {
		t.Fatalf("UpdateStatusByAuthorizationID failed: %v", err)
	}
	if err := notifier.BroadcastSubastaCerrada(ctx, auctionID, &bid); err != nil {
		t.Fatalf("BroadcastSubastaCerrada failed: %v", err)
	}
}
