package polymarket

import (
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestOrderToJSONV2UsesNumericSaltAndV2Fields(t *testing.T) {
	order := &SignedOrder{
		Salt:          big.NewInt(12345),
		Maker:         common.HexToAddress("0x1111111111111111111111111111111111111111"),
		Signer:        common.HexToAddress("0x2222222222222222222222222222222222222222"),
		TokenId:       big.NewInt(98765),
		MakerAmount:   big.NewInt(1000000),
		TakerAmount:   big.NewInt(2000000),
		Expiration:    big.NewInt(0),
		Side:          big.NewInt(0),
		SignatureType: big.NewInt(SignatureType1271),
		Timestamp:     big.NewInt(1710000000000),
		Metadata:      "0x0000000000000000000000000000000000000000000000000000000000000000",
		Builder:       "0x0000000000000000000000000000000000000000000000000000000000000000",
		Signature:     []byte("0xabc"),
		Version:       2,
	}

	payload := OrderToJSONWithPostOnly(order, "api-key", OrderTypeGTC, true)
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	bodyStr := string(body)

	if !strings.Contains(bodyStr, `"salt":12345`) {
		t.Fatalf("expected numeric salt in body, got %s", bodyStr)
	}
	if strings.Contains(bodyStr, "feeRateBps") || strings.Contains(bodyStr, "nonce") {
		t.Fatalf("v2 body should not contain v1-only fields: %s", bodyStr)
	}

	orderPayload := payload["order"].(map[string]interface{})
	if orderPayload["signatureType"] != SignatureType1271 {
		t.Fatalf("signatureType mismatch: got %v", orderPayload["signatureType"])
	}
	if orderPayload["timestamp"] != "1710000000000" {
		t.Fatalf("timestamp mismatch: got %v", orderPayload["timestamp"])
	}
}
