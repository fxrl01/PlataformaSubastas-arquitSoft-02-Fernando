package persistence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresBidRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresBidRepository(pool *pgxpool.Pool) *PostgresBidRepository {
	return &PostgresBidRepository{pool: pool}
}

func (r *PostgresBidRepository) BuscarPorIdempotencyKey(ctx context.Context, subastaID, bidderID, key string) (*domain.Puja, error) {
	var bid domain.Puja
	var amount string
	err := r.pool.QueryRow(ctx, `
		SELECT id, subasta_id, postor_id, monto::text, timestamp_oficial
		FROM pujas
		WHERE subasta_id = $1 AND postor_id = $2 AND idempotency_key = $3`,
		subastaID, bidderID, key).Scan(
		&bid.ID, &bid.SubastaID, &bid.PostorID, &amount, &bid.Timestamp,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find bid by idempotency key: %w", err)
	}
	bid.Monto, err = strconv.ParseFloat(amount, 64)
	if err != nil {
		return nil, fmt.Errorf("parse idempotent bid amount: %w", err)
	}
	return &bid, nil
}

func (r *PostgresBidRepository) RegistrarPujaAtomica(ctx context.Context, subastaID string, puja domain.Puja, precioAnterior float64, fechaCierre time.Time, tiempoExtendido bool, idempotencyKey string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin bid transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var estado string
	var precioActual string
	var postorLiderID *string
	err = tx.QueryRow(ctx, `
		SELECT estado, precio_actual::text, postor_lider_id
		FROM subastas WHERE id = $1 FOR UPDATE`, subastaID).Scan(&estado, &precioActual, &postorLiderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrSubastaNoEncontrada
	}
	if err != nil {
		return fmt.Errorf("lock auction: %w", err)
	}
	if estado != string(domain.SubastaActiva) && estado != string(domain.SubastaEnConteo) {
		return domain.ErrSubastaCerrada
	}
	if postorLiderID != nil && *postorLiderID == puja.PostorID {
		return domain.ErrPostorIgual
	}
	precioActualValue, err := strconv.ParseFloat(precioActual, 64)
	if err != nil {
		return fmt.Errorf("parse auction price: %w", err)
	}
	if precioActualValue != precioAnterior {
		return domain.ErrPujaConcurrente
	}

	var correlativo int
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(correlativo), 0) + 1
		FROM pujas WHERE subasta_id = $1`, subastaID).Scan(&correlativo)
	if err != nil {
		return fmt.Errorf("next bid sequence: %w", err)
	}

	var prevHash *string
	_ = tx.QueryRow(ctx, `
		SELECT hash_actual FROM logs_auditoria
		WHERE subasta_id = $1 ORDER BY id DESC LIMIT 1`, subastaID).Scan(&prevHash)

	prevHashVal := "GENESIS"
	if prevHash != nil && *prevHash != "" {
		prevHashVal = *prevHash
	}
	rawDigest := fmt.Sprintf("%s|%s|%s|%s|%.2f|%d|%d",
		prevHashVal, subastaID, puja.ID, puja.PostorID, puja.Monto, correlativo, puja.Timestamp.UTC().UnixNano())
	sum := sha256.Sum256([]byte(rawDigest))
	auditHash := hex.EncodeToString(sum[:])

	_, err = tx.Exec(ctx, `
		UPDATE subastas
		SET precio_actual = $1, postor_lider_id = $2, fin_programado = $3
		WHERE id = $4`,
		puja.Monto, puja.PostorID, fechaCierre, subastaID)
	if err != nil {
		return fmt.Errorf("update auction price: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO pujas
			(id, subasta_id, postor_id, monto, correlativo, timestamp_oficial, anti_sniping_disparado, idempotency_key, hash_auditoria)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''), $9)`,
		puja.ID, subastaID, puja.PostorID, puja.Monto, correlativo, puja.Timestamp, tiempoExtendido, idempotencyKey, auditHash)
	if err != nil {
		return fmt.Errorf("insert bid: %w", err)
	}

	auditPayload, _ := json.Marshal(map[string]any{
		"puja_id":          puja.ID,
		"postor_id":        puja.PostorID,
		"monto":            puja.Monto,
		"correlativo":      correlativo,
		"tiempo_extendido": tiempoExtendido,
	})
	_, err = tx.Exec(ctx, `
		INSERT INTO logs_auditoria (subasta_id, evento, payload, hash_anterior, hash_actual)
		VALUES ($1, 'PUJA_REGISTRADA', $2, $3, $4)`,
		subastaID, auditPayload, prevHash, auditHash)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit bid transaction: %w", err)
	}
	return nil
}

func (r *PostgresBidRepository) ListarPorSubasta(ctx context.Context, subastaID, bidderID string, minimumAmount float64, limit, offset int) ([]domain.Puja, int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, subasta_id, postor_id, monto::text, timestamp_oficial
		FROM pujas
		WHERE subasta_id = $1
		  AND ($2 = '' OR postor_id = $2)
		  AND ($3 = 0 OR monto >= $3)
		ORDER BY correlativo ASC
		LIMIT $4 OFFSET $5`, subastaID, bidderID, minimumAmount, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list auction bids: %w", err)
	}
	defer rows.Close()

	bids := make([]domain.Puja, 0)
	for rows.Next() {
		var bid domain.Puja
		var amount string
		if err := rows.Scan(&bid.ID, &bid.SubastaID, &bid.PostorID, &amount, &bid.Timestamp); err != nil {
			return nil, 0, fmt.Errorf("scan auction bid: %w", err)
		}
		bid.Monto, err = strconv.ParseFloat(amount, 64)
		if err != nil {
			return nil, 0, fmt.Errorf("parse bid amount: %w", err)
		}
		bids = append(bids, bid)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate auction bids: %w", err)
	}
	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM pujas
		WHERE subasta_id = $1
		  AND ($2 = '' OR postor_id = $2)
		  AND ($3 = 0 OR monto >= $3)`,
		subastaID, bidderID, minimumAmount).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count auction bids: %w", err)
	}
	return bids, total, nil
}

func (r *PostgresBidRepository) ObtenerHashesAuditoria(ctx context.Context, subastaID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, postor_id, monto::text, correlativo, timestamp_oficial, COALESCE(hash_auditoria, '')
		FROM pujas
		WHERE subasta_id = $1
		ORDER BY correlativo ASC`, subastaID)
	if err != nil {
		return nil, fmt.Errorf("list bid audit hashes: %w", err)
	}
	defer rows.Close()

	hashes := make([]string, 0)
	for rows.Next() {
		var id, postorID, amountStr, auditHash string
		var correlativo int
		var ts time.Time
		if err := rows.Scan(&id, &postorID, &amountStr, &correlativo, &ts, &auditHash); err != nil {
			return nil, fmt.Errorf("scan bid audit hash: %w", err)
		}
		if auditHash == "" {
			raw := fmt.Sprintf("LEGACY|%s|%s|%s|%s|%d|%d", subastaID, id, postorID, amountStr, correlativo, ts.UTC().UnixNano())
			sum := sha256.Sum256([]byte(raw))
			auditHash = hex.EncodeToString(sum[:])
		}
		hashes = append(hashes, auditHash)
	}
	return hashes, rows.Err()
}

func (r *PostgresBidRepository) ObtenerCadenaAuditoria(ctx context.Context, subastaID string) ([]domain.AuditLogEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT evento, COALESCE(hash_anterior, 'GENESIS'), hash_actual
		FROM logs_auditoria
		WHERE subasta_id = $1
		ORDER BY id ASC`, subastaID)
	if err != nil {
		return nil, fmt.Errorf("list audit chain: %w", err)
	}
	defer rows.Close()

	entries := make([]domain.AuditLogEntry, 0)
	for rows.Next() {
		var entry domain.AuditLogEntry
		if err := rows.Scan(&entry.Event, &entry.PreviousHash, &entry.CurrentHash); err != nil {
			return nil, fmt.Errorf("scan audit chain entry: %w", err)
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (r *PostgresBidRepository) GuardarReciboOnChain(ctx context.Context, receipt domain.OnChainSettlementReceipt) error {
	var prevHash *string
	_ = r.pool.QueryRow(ctx, `
		SELECT hash_actual FROM logs_auditoria
		WHERE subasta_id = $1 ORDER BY id DESC LIMIT 1`, receipt.AuctionID).Scan(&prevHash)

	payload, err := json.Marshal(receipt)
	if err != nil {
		return fmt.Errorf("marshal onchain receipt: %w", err)
	}
	txHashNormalized := receipt.TxHash
	if len(txHashNormalized) > 64 {
		txHashNormalized = txHashNormalized[len(txHashNormalized)-64:]
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO logs_auditoria (subasta_id, evento, payload, hash_anterior, hash_actual)
		VALUES ($1, 'CIERRE_ONCHAIN_MERKLE', $2, $3, $4)`,
		receipt.AuctionID, payload, prevHash, txHashNormalized)
	if err != nil {
		return fmt.Errorf("save onchain settlement receipt: %w", err)
	}
	return nil
}

func (r *PostgresBidRepository) ObtenerReciboOnChain(ctx context.Context, subastaID string) (*domain.OnChainSettlementReceipt, error) {
	var payload []byte
	err := r.pool.QueryRow(ctx, `
		SELECT payload FROM logs_auditoria
		WHERE subasta_id = $1 AND evento = 'CIERRE_ONCHAIN_MERKLE'
		ORDER BY id DESC LIMIT 1`, subastaID).Scan(&payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query onchain receipt: %w", err)
	}
	var receipt domain.OnChainSettlementReceipt
	if err := json.Unmarshal(payload, &receipt); err != nil {
		return nil, fmt.Errorf("unmarshal onchain receipt: %w", err)
	}
	return &receipt, nil
}
