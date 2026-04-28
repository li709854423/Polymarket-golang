package order_builder

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/polymarket/go-order-utils/pkg/model"
)

const (
	bytes32Zero = "0x0000000000000000000000000000000000000000000000000000000000000000"
)

// SignedOrder is the SDK order representation. It keeps the v1 fields used by
// older callers and adds the CLOB v2 fields required by the current API.
type SignedOrder struct {
	Salt          *big.Int
	Maker         common.Address
	Signer        common.Address
	Taker         common.Address
	TokenId       *big.Int
	MakerAmount   *big.Int
	TakerAmount   *big.Int
	Expiration    *big.Int
	Nonce         *big.Int
	FeeRateBps    *big.Int
	Side          *big.Int
	SignatureType *big.Int
	Signature     []byte

	Timestamp *big.Int
	Metadata  string
	Builder   string
	Version   int
}

// OrderDataV2 contains the fields signed by the CLOB v2 exchange.
type OrderDataV2 struct {
	Maker         string
	Signer        string
	TokenId       string
	MakerAmount   string
	TakerAmount   string
	Side          model.Side
	SignatureType int
	Timestamp     string
	Metadata      string
	Builder       string
	Expiration    string
}

func signedOrderFromV1(order *model.SignedOrder) *SignedOrder {
	if order == nil {
		return nil
	}

	return &SignedOrder{
		Salt:          order.Salt,
		Maker:         order.Maker,
		Signer:        order.Signer,
		Taker:         order.Taker,
		TokenId:       order.TokenId,
		MakerAmount:   order.MakerAmount,
		TakerAmount:   order.TakerAmount,
		Expiration:    order.Expiration,
		Nonce:         order.Nonce,
		FeeRateBps:    order.FeeRateBps,
		Side:          order.Side,
		SignatureType: order.SignatureType,
		Signature:     order.Signature,
		Version:       1,
	}
}

// BuildSignedOrderV2 builds and signs a CLOB v2 order using the private key
// signer, while allowing maker/funder to be a proxy, Safe, or 1271 account.
func (ob *OrderBuilder) BuildSignedOrderV2(orderData *OrderDataV2, exchangeAddr string, chainID int) (*SignedOrder, error) {
	if orderData == nil {
		return nil, fmt.Errorf("order data is required")
	}

	if !strings.EqualFold(orderData.Signer, ob.signer.Address()) {
		return nil, fmt.Errorf("signer does not match private key address")
	}

	salt, err := generateOrderSalt()
	if err != nil {
		return nil, err
	}

	timestamp := orderData.Timestamp
	if timestamp == "" {
		timestamp = strconvFormatInt(time.Now().UnixMilli())
	}

	metadata := normalizeBytes32(orderData.Metadata)
	builder := normalizeBytes32(orderData.Builder)

	tokenID, err := parseBig(orderData.TokenId, "tokenId")
	if err != nil {
		return nil, err
	}
	makerAmount, err := parseBig(orderData.MakerAmount, "makerAmount")
	if err != nil {
		return nil, err
	}
	takerAmount, err := parseBig(orderData.TakerAmount, "takerAmount")
	if err != nil {
		return nil, err
	}
	expiration, err := parseBig(defaultString(orderData.Expiration, "0"), "expiration")
	if err != nil {
		return nil, err
	}
	timestampBig, err := parseBig(timestamp, "timestamp")
	if err != nil {
		return nil, err
	}

	order := &SignedOrder{
		Salt:          salt,
		Maker:         common.HexToAddress(orderData.Maker),
		Signer:        common.HexToAddress(orderData.Signer),
		Taker:         common.Address{},
		TokenId:       tokenID,
		MakerAmount:   makerAmount,
		TakerAmount:   takerAmount,
		Expiration:    expiration,
		Nonce:         big.NewInt(0),
		FeeRateBps:    big.NewInt(0),
		Side:          big.NewInt(int64(orderData.Side)),
		SignatureType: big.NewInt(int64(orderData.SignatureType)),
		Timestamp:     timestampBig,
		Metadata:      metadata,
		Builder:       builder,
		Version:       2,
	}

	digest, err := buildV2OrderDigest(order, exchangeAddr, chainID)
	if err != nil {
		return nil, err
	}

	signature, err := ob.signer.Sign(digest.Bytes())
	if err != nil {
		return nil, err
	}
	order.Signature = []byte(signature)

	return order, nil
}

func generateOrderSalt() (*big.Int, error) {
	max := big.NewInt(time.Now().UnixMilli())
	if max.Sign() <= 0 {
		max = big.NewInt(1)
	}
	return rand.Int(rand.Reader, max)
}

func buildV2OrderDigest(order *SignedOrder, exchangeAddr string, chainID int) (common.Hash, error) {
	domainSeparator := buildV2DomainSeparator(exchangeAddr, chainID)
	structHash, err := buildV2OrderStructHash(order)
	if err != nil {
		return common.Hash{}, err
	}

	signable := append([]byte{0x19, 0x01}, domainSeparator.Bytes()...)
	signable = append(signable, structHash.Bytes()...)
	return crypto.Keccak256Hash(signable), nil
}

func buildV2DomainSeparator(exchangeAddr string, chainID int) common.Hash {
	typeHash := crypto.Keccak256Hash([]byte("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"))
	nameHash := crypto.Keccak256Hash([]byte("Polymarket CTF Exchange"))
	versionHash := crypto.Keccak256Hash([]byte("2"))

	encoded := make([]byte, 0, 32*5)
	encoded = append(encoded, typeHash.Bytes()...)
	encoded = append(encoded, nameHash.Bytes()...)
	encoded = append(encoded, versionHash.Bytes()...)
	encoded = append(encoded, uint256Bytes(big.NewInt(int64(chainID)))...)
	encoded = append(encoded, addressBytes(common.HexToAddress(exchangeAddr))...)
	return crypto.Keccak256Hash(encoded)
}

func buildV2OrderStructHash(order *SignedOrder) (common.Hash, error) {
	typeHash := crypto.Keccak256Hash([]byte("Order(uint256 salt,address maker,address signer,uint256 tokenId,uint256 makerAmount,uint256 takerAmount,uint8 side,uint8 signatureType,uint256 timestamp,bytes32 metadata,bytes32 builder)"))

	metadata, err := bytes32Bytes(order.Metadata)
	if err != nil {
		return common.Hash{}, fmt.Errorf("invalid metadata: %w", err)
	}
	builder, err := bytes32Bytes(order.Builder)
	if err != nil {
		return common.Hash{}, fmt.Errorf("invalid builder: %w", err)
	}

	encoded := make([]byte, 0, 32*12)
	encoded = append(encoded, typeHash.Bytes()...)
	encoded = append(encoded, uint256Bytes(order.Salt)...)
	encoded = append(encoded, addressBytes(order.Maker)...)
	encoded = append(encoded, addressBytes(order.Signer)...)
	encoded = append(encoded, uint256Bytes(order.TokenId)...)
	encoded = append(encoded, uint256Bytes(order.MakerAmount)...)
	encoded = append(encoded, uint256Bytes(order.TakerAmount)...)
	encoded = append(encoded, uint256Bytes(order.Side)...)
	encoded = append(encoded, uint256Bytes(order.SignatureType)...)
	encoded = append(encoded, uint256Bytes(order.Timestamp)...)
	encoded = append(encoded, metadata...)
	encoded = append(encoded, builder...)

	return crypto.Keccak256Hash(encoded), nil
}

func uint256Bytes(v *big.Int) []byte {
	if v == nil {
		v = big.NewInt(0)
	}
	return common.LeftPadBytes(v.Bytes(), 32)
}

func addressBytes(addr common.Address) []byte {
	return common.LeftPadBytes(addr.Bytes(), 32)
}

func bytes32Bytes(value string) ([]byte, error) {
	value = strings.TrimPrefix(normalizeBytes32(value), "0x")
	decoded, err := hex.DecodeString(value)
	if err != nil {
		return nil, err
	}
	if len(decoded) != 32 {
		return nil, fmt.Errorf("expected 32 bytes, got %d", len(decoded))
	}
	return decoded, nil
}

func normalizeBytes32(value string) string {
	if value == "" {
		return bytes32Zero
	}
	return value
}

func parseBig(value, field string) (*big.Int, error) {
	if value == "" {
		return big.NewInt(0), nil
	}
	n, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return nil, fmt.Errorf("invalid %s: %s", field, value)
	}
	return n, nil
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func strconvFormatInt(v int64) string {
	return new(big.Int).SetInt64(v).String()
}
