package web3_test

import (
	"context"
	"testing"
	"time"

	"github.com/fxrl01/livebid-backend/internal/adapters/web3"
	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

func TestMerkleTreeRootAndProofVerification(t *testing.T) {
	leaves := []string{
		"a111111111111111111111111111111111111111111111111111111111111111",
		"b222222222222222222222222222222222222222222222222222222222222222",
		"c333333333333333333333333333333333333333333333333333333333333333",
		"d444444444444444444444444444444444444444444444444444444444444444",
		"e555555555555555555555555555555555555555555555555555555555555555",
	}

	root := domain.ComputeMerkleRoot(leaves)
	if len(root) != 66 || root[:2] != "0x" {
		t.Fatalf("expected 0x-prefixed 32-byte Merkle root, got %s", root)
	}

	for idx, leaf := range leaves {
		proof, ok := domain.GenerateMerkleProof(leaves, idx)
		if !ok {
			t.Fatalf("failed to generate proof for leaf %d", idx)
		}
		if !domain.VerifyMerkleProof(leaf, root, proof) {
			t.Fatalf("valid leaf %d failed Merkle proof verification", idx)
		}
		if domain.VerifyMerkleProof("f999999999999999999999999999999999999999999999999999999999999999", root, proof) {
			t.Fatalf("tampered leaf should not pass Merkle proof verification at index %d", idx)
		}
	}
}

func TestEIP712VerifierAcceptsValidSignedBidAndRejectsTampering(t *testing.T) {
	verifier := web3.NewEIP712Verifier(8453, "0x1111111111111111111111111111111111111111")
	now := time.Now()

	env := domain.EIP712BidEnvelope{
		AuctionID:     "SUB-2026-001",
		BidderID:      "usr-vip-01",
		WalletAddress: "0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
		Amount:        48000.00,
		Nonce:         "nonce-001",
		Deadline:      now.Add(5 * time.Minute).Unix(),
		ChainID:       8453,
	}
	env.Signature = verifier.SignBidEnvelope(env)

	if err := verifier.VerifyBidSignature(context.Background(), env, now); err != nil {
		t.Fatalf("expected valid EIP-712 signature, got %v", err)
	}

	// Tamper with amount
	tampered := env
	tampered.Amount = 49000.00
	if err := verifier.VerifyBidSignature(context.Background(), tampered, now); err != domain.ErrInvalidWeb3Signature {
		t.Fatalf("expected ErrInvalidWeb3Signature for tampered amount, got %v", err)
	}

	// Expired deadline
	if err := verifier.VerifyBidSignature(context.Background(), env, now.Add(10*time.Minute)); err != domain.ErrInvalidWeb3Signature {
		t.Fatalf("expected ErrInvalidWeb3Signature for expired deadline, got %v", err)
	}
}

func TestEVMEscrowAdapterLocksCollateralAndAnchorsMerkleRoot(t *testing.T) {
	escrow := web3.NewEVMEscrowAdapter(8453, "0x1111111111111111111111111111111111111111", "")

	txHash, err := escrow.LockCollateralOnChain(
		context.Background(),
		"SUB-1",
		"usr-1",
		"0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
		4500,
	)
	if err != nil || len(txHash) != 66 {
		t.Fatalf("expected valid lock txHash, got tx=%s err=%v", txHash, err)
	}

	root := domain.ComputeMerkleRoot([]string{"abc123", "def456"})
	receipt, err := escrow.SettleEscrowAndAnchorMerkleRoot(context.Background(), "SUB-1", "usr-1", 52000, root)
	if err != nil {
		t.Fatalf("unexpected settlement error: %v", err)
	}
	if receipt.MerkleRoot != root || len(receipt.TxHash) != 66 || receipt.ChainID != 8453 {
		t.Fatalf("unexpected on-chain settlement receipt: %+v", receipt)
	}
}

