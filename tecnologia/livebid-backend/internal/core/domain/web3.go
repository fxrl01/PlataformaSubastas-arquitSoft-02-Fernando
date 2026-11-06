package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

// EIP712BidEnvelope representa una oferta firmada criptográficamente bajo el estándar EIP-712.
type EIP712BidEnvelope struct {
	AuctionID     string  `json:"auction_id"`
	BidderID      string  `json:"bidder_id"`
	WalletAddress string  `json:"wallet_address"`
	Amount        float64 `json:"amount"`
	Nonce         string  `json:"nonce"`
	Deadline      int64   `json:"deadline"`
	Signature     string  `json:"signature"`
	ChainID       int64   `json:"chain_id"`
}

// OnChainSettlementReceipt contiene el comprobante inmutable del anclaje en Smart Contract al cerrar la subasta.
type OnChainSettlementReceipt struct {
	AuctionID       string    `json:"subasta_id"`
	MerkleRoot      string    `json:"merkle_root"`
	WinnerID        string    `json:"ganador_id,omitempty"`
	WinningAmount   float64   `json:"monto_adjudicado,omitempty"`
	TxHash          string    `json:"tx_hash"`
	BlockNumber     uint64    `json:"block_number"`
	ChainID         int64     `json:"chain_id"`
	ContractAddress string    `json:"contract_address"`
	AnchoredAt      time.Time `json:"anchored_at"`
}

// MerkleProofStep representa un nodo hermano en la ruta de verificación de un árbol de Merkle.
type MerkleProofStep struct {
	Position string `json:"position"` // "LEFT" o "RIGHT"
	Hash     string `json:"hash"`
}

// IsValidWalletAddress verifica que la dirección cumpla con el formato hexadecimal EVM (0x + 40 hex chars).
func IsValidWalletAddress(address string) bool {
	address = strings.TrimSpace(address)
	if len(address) != 42 || !strings.HasPrefix(address, "0x") && !strings.HasPrefix(address, "0X") {
		return false
	}
	_, err := hex.DecodeString(address[2:])
	return err == nil
}

// ComputeMerkleRoot calcula la raíz de Merkle determinista (0x...) a partir de los hashes SHA-256 de las pujas.
func ComputeMerkleRoot(leafHashes []string) string {
	if len(leafHashes) == 0 {
		empty := sha256.Sum256([]byte("LIVEBID_EMPTY_AUCTION_TREE"))
		return "0x" + hex.EncodeToString(empty[:])
	}

	level := make([]string, len(leafHashes))
	for i, h := range leafHashes {
		level[i] = normalizeHex(h)
	}

	for len(level) > 1 {
		if len(level)%2 != 0 {
			level = append(level, level[len(level)-1])
		}
		nextLevel := make([]string, 0, len(level)/2)
		for i := 0; i < len(level); i += 2 {
			nextLevel = append(nextLevel, hashPair(level[i], level[i+1]))
		}
		level = nextLevel
	}

	return "0x" + level[0]
}

// GenerateMerkleProof genera la ruta criptográfica de inclusión para una puja en el índice dado.
func GenerateMerkleProof(leafHashes []string, index int) ([]MerkleProofStep, bool) {
	if index < 0 || index >= len(leafHashes) {
		return nil, false
	}

	level := make([]string, len(leafHashes))
	for i, h := range leafHashes {
		level[i] = normalizeHex(h)
	}

	proof := make([]MerkleProofStep, 0)
	currIndex := index

	for len(level) > 1 {
		if len(level)%2 != 0 {
			level = append(level, level[len(level)-1])
		}
		if currIndex%2 == 0 {
			proof = append(proof, MerkleProofStep{
				Position: "RIGHT",
				Hash:     "0x" + level[currIndex+1],
			})
		} else {
			proof = append(proof, MerkleProofStep{
				Position: "LEFT",
				Hash:     "0x" + level[currIndex-1],
			})
		}

		nextLevel := make([]string, 0, len(level)/2)
		for i := 0; i < len(level); i += 2 {
			nextLevel = append(nextLevel, hashPair(level[i], level[i+1]))
		}
		level = nextLevel
		currIndex /= 2
	}

	return proof, true
}

// VerifyMerkleProof comprueba que una hoja pertenezca al Merkle Root dado su camino de prueba.
func VerifyMerkleProof(leafHash, expectedRoot string, proof []MerkleProofStep) bool {
	if leafHash == "" || expectedRoot == "" {
		return false
	}
	curr := normalizeHex(leafHash)
	for _, step := range proof {
		sibling := normalizeHex(step.Hash)
		if step.Position == "LEFT" {
			curr = hashPair(sibling, curr)
		} else {
			curr = hashPair(curr, sibling)
		}
	}
	return "0x"+curr == "0x"+normalizeHex(expectedRoot)
}

func hashPair(left, right string) string {
	sum := sha256.Sum256([]byte(left + ":" + right))
	return hex.EncodeToString(sum[:])
}

func normalizeHex(h string) string {
	h = strings.TrimSpace(strings.ToLower(h))
	h = strings.TrimPrefix(h, "0x")
	return h
}

