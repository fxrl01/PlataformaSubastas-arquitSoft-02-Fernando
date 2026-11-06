package web3

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

// EIP712Verifier implementa domain.Web3SignatureVerifierPort siguiendo la estructura de datos tipados EIP-712.
type EIP712Verifier struct {
	domainName        string
	domainVersion     string
	defaultChainID    int64
	verifyingContract string
}

func NewEIP712Verifier(chainID int64, verifyingContract string) *EIP712Verifier {
	if chainID <= 0 {
		chainID = 8453 // Base L2 por defecto
	}
	if !domain.IsValidWalletAddress(verifyingContract) {
		verifyingContract = "0x000000000000000000000000000000000000dEaD"
	}
	return &EIP712Verifier{
		domainName:        "LiveBid Protocol",
		domainVersion:     "1",
		defaultChainID:    chainID,
		verifyingContract: strings.ToLower(verifyingContract),
	}
}

func (v *EIP712Verifier) ComputeTypedDataDigest(env domain.EIP712BidEnvelope) string {
	chainID := env.ChainID
	if chainID <= 0 {
		chainID = v.defaultChainID
	}
	domainSepInput := fmt.Sprintf(
		"EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)|%s|%s|%d|%s",
		v.domainName, v.domainVersion, chainID, v.verifyingContract,
	)
	domainSep := sha256.Sum256([]byte(domainSepInput))

	amountCents := int64(math.Round(env.Amount * 100))
	structInput := fmt.Sprintf(
		"Bid(string auctionId,string bidderId,address walletAddress,uint256 amountCents,string nonce,uint256 deadline)|%s|%s|%s|%d|%s|%d",
		env.AuctionID,
		env.BidderID,
		strings.ToLower(strings.TrimSpace(env.WalletAddress)),
		amountCents,
		env.Nonce,
		env.Deadline,
	)
	structHash := sha256.Sum256([]byte(structInput))

	raw := make([]byte, 0, 2+len(domainSep)+len(structHash))
	raw = append(raw, 0x19, 0x01)
	raw = append(raw, domainSep[:]...)
	raw = append(raw, structHash[:]...)
	digest := sha256.Sum256(raw)
	return "0x" + hex.EncodeToString(digest[:])
}

// SignBidEnvelope firma un sobre EIP-712 usando la dirección de wallet como clave de prueba determinista.
func (v *EIP712Verifier) SignBidEnvelope(env domain.EIP712BidEnvelope) string {
	digest := v.ComputeTypedDataDigest(env)
	mac := hmac.New(sha256.New, []byte(strings.ToLower(strings.TrimSpace(env.WalletAddress))))
	_, _ = mac.Write([]byte(digest))
	return "0x" + hex.EncodeToString(mac.Sum(nil))
}

func (v *EIP712Verifier) VerifyBidSignature(ctx context.Context, env domain.EIP712BidEnvelope, now time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !domain.IsValidWalletAddress(env.WalletAddress) {
		return domain.ErrInvalidWeb3Wallet
	}
	if env.Deadline > 0 && now.Unix() > env.Deadline {
		return domain.ErrInvalidWeb3Signature
	}
	sig := strings.TrimSpace(env.Signature)
	if !strings.HasPrefix(sig, "0x") && !strings.HasPrefix(sig, "0X") {
		return domain.ErrInvalidWeb3Signature
	}
	expected := v.SignBidEnvelope(env)
	if !hmac.Equal([]byte(strings.ToLower(sig)), []byte(strings.ToLower(expected))) {
		return domain.ErrInvalidWeb3Signature
	}
	return nil
}

