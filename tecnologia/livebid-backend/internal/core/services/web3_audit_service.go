package services

import (
	"context"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

type Web3AuditService struct {
	auctions  domain.SubastaRepository
	auditRepo domain.AuditTrailRepository
}

type MerkleProofView struct {
	LeafIndex int                      `json:"indice_hoja"`
	LeafHash  string                   `json:"hash_hoja"`
	Proof     []domain.MerkleProofStep `json:"ruta_merkle"`
	Verified  bool                     `json:"verificada"`
}

type Web3AuditReport struct {
	AuctionID        string                          `json:"subasta_id"`
	Status           domain.EstadoSubasta            `json:"estado"`
	MerkleRoot       string                          `json:"merkle_root"`
	TotalAuditedBids int                             `json:"total_pujas_auditadas"`
	ChainIntact      bool                            `json:"cadena_integra"`
	LeafHashes       []string                        `json:"hashes_hojas"`
	AuditChain       []domain.AuditLogEntry          `json:"cadena_eventos"`
	OnChainReceipt   *domain.OnChainSettlementReceipt `json:"recibo_onchain,omitempty"`
	MerkleProof      *MerkleProofView                `json:"prueba_merkle,omitempty"`
}

func NewWeb3AuditService(auctions domain.SubastaRepository, auditRepo domain.AuditTrailRepository) *Web3AuditService {
	return &Web3AuditService{auctions: auctions, auditRepo: auditRepo}
}

func (s *Web3AuditService) InspectAuction(ctx context.Context, auctionID string, proofIndex *int) (Web3AuditReport, error) {
	subasta, err := s.auctions.ObtenerPorID(ctx, auctionID)
	if err != nil {
		return Web3AuditReport{}, err
	}
	hashes, err := s.auditRepo.ObtenerHashesAuditoria(ctx, auctionID)
	if err != nil {
		return Web3AuditReport{}, err
	}
	chain, err := s.auditRepo.ObtenerCadenaAuditoria(ctx, auctionID)
	if err != nil {
		return Web3AuditReport{}, err
	}
	receipt, err := s.auditRepo.ObtenerReciboOnChain(ctx, auctionID)
	if err != nil {
		return Web3AuditReport{}, err
	}

	merkleRoot := domain.ComputeMerkleRoot(hashes)
	formattedLeaves := make([]string, len(hashes))
	for i, h := range hashes {
		if len(h) >= 2 && h[:2] == "0x" {
			formattedLeaves[i] = h
		} else {
			formattedLeaves[i] = "0x" + h
		}
	}

	report := Web3AuditReport{
		AuctionID:        subasta.ID,
		Status:           subasta.Estado,
		MerkleRoot:       merkleRoot,
		TotalAuditedBids: len(hashes),
		ChainIntact:      verifyHashChain(chain),
		LeafHashes:       formattedLeaves,
		AuditChain:       chain,
		OnChainReceipt:   receipt,
	}

	if proofIndex != nil {
		idx := *proofIndex
		steps, ok := domain.GenerateMerkleProof(hashes, idx)
		if ok {
			report.MerkleProof = &MerkleProofView{
				LeafIndex: idx,
				LeafHash:  formattedLeaves[idx],
				Proof:     steps,
				Verified:  domain.VerifyMerkleProof(hashes[idx], merkleRoot, steps),
			}
		}
	}

	return report, nil
}

func verifyHashChain(entries []domain.AuditLogEntry) bool {
	if len(entries) == 0 {
		return true
	}
	expectedPrev := "GENESIS"
	for i, entry := range entries {
		if entry.CurrentHash == "" {
			return false
		}
		if i == 0 {
			if entry.PreviousHash != "" && entry.PreviousHash != "GENESIS" {
				return false
			}
		} else if entry.PreviousHash != expectedPrev {
			return false
		}
		expectedPrev = entry.CurrentHash
	}
	return true
}

