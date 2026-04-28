package polymarket

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// ParseRawOrderBookSummary 解析原始订单簿摘要
func ParseRawOrderBookSummary(rawObs map[string]interface{}) (*OrderBookSummary, error) {
	bids := []OrderSummary{}
	if bidsRaw, ok := rawObs["bids"].([]interface{}); ok {
		for _, bidRaw := range bidsRaw {
			if bid, ok := bidRaw.(map[string]interface{}); ok {
				bids = append(bids, OrderSummary{
					Price: fmt.Sprintf("%v", bid["price"]),
					Size:  fmt.Sprintf("%v", bid["size"]),
				})
			}
		}
	}

	asks := []OrderSummary{}
	if asksRaw, ok := rawObs["asks"].([]interface{}); ok {
		for _, askRaw := range asksRaw {
			if ask, ok := askRaw.(map[string]interface{}); ok {
				asks = append(asks, OrderSummary{
					Price: fmt.Sprintf("%v", ask["price"]),
					Size:  fmt.Sprintf("%v", ask["size"]),
				})
			}
		}
	}

	obs := &OrderBookSummary{
		Market:       getString(rawObs, "market"),
		AssetID:      getString(rawObs, "asset_id"),
		Timestamp:    getString(rawObs, "timestamp"),
		MinOrderSize: getString(rawObs, "min_order_size"),
		NegRisk:      getBool(rawObs, "neg_risk"),
		TickSize:     getString(rawObs, "tick_size"),
		Bids:         bids,
		Asks:         asks,
		Hash:         getString(rawObs, "hash"),
	}

	return obs, nil
}

// GenerateOrderBookSummaryHash 生成订单簿摘要哈希
func GenerateOrderBookSummaryHash(orderbook *OrderBookSummary) string {
	// 临时清空hash
	originalHash := orderbook.Hash
	orderbook.Hash = ""

	// 序列化为JSON
	jsonData, err := json.Marshal(orderbook)
	if err != nil {
		orderbook.Hash = originalHash
		return ""
	}

	// SHA1哈希
	hash := sha1.Sum(jsonData)
	hashStr := fmt.Sprintf("%x", hash)

	// 恢复hash
	orderbook.Hash = hashStr
	return hashStr
}

// OrderToJSON 将订单转换为JSON格式
// 格式与 Python py_order_utils.SignedOrder.dict() 完全一致
func OrderToJSON(order *SignedOrder, owner string, orderType OrderType) map[string]interface{} {
	return OrderToJSONWithPostOnly(order, owner, orderType, false)
}

// OrderToJSONWithPostOnly 将订单转换为JSON格式（支持 PostOnly）
func OrderToJSONWithPostOnly(order *SignedOrder, owner string, orderType OrderType, postOnly bool) map[string]interface{} {
	// 将签名从 []byte 转换为 hex 字符串（带 0x 前缀）
	signatureHex := signatureToHex(order.Signature)

	// 将 side 从数字转换为字符串 "BUY" 或 "SELL"
	// Python: BUY = 0, SELL = 1
	sideStr := "BUY"
	if order.Side != nil && order.Side.Int64() == 1 {
		sideStr = "SELL"
	}

	if order.Version == 2 {
		orderDict := map[string]interface{}{
			"salt":          bigIntInt64(order.Salt), // CLOB v2 requires a JSON number
			"maker":         common.HexToAddress(order.Maker.Hex()).Hex(),
			"signer":        common.HexToAddress(order.Signer.Hex()).Hex(),
			"tokenId":       bigIntString(order.TokenId),
			"makerAmount":   bigIntString(order.MakerAmount),
			"takerAmount":   bigIntString(order.TakerAmount),
			"side":          sideStr,
			"signatureType": int(bigIntInt64(order.SignatureType)),
			"timestamp":     bigIntString(order.Timestamp),
			"expiration":    bigIntString(order.Expiration),
			"metadata":      order.Metadata,
			"builder":       order.Builder,
			"signature":     signatureHex,
		}
		return map[string]interface{}{
			"order":     orderDict,
			"owner":     owner,
			"orderType": string(orderType),
			"postOnly":  postOnly,
			"deferExec": false,
		}
	}

	// 将SignedOrder转换为字典
	// 格式与 Python py_order_utils.SignedOrder.dict() 完全一致
	orderDict := map[string]interface{}{
		"salt":          bigIntInt64(order.Salt), // 整数，不是字符串
		"maker":         common.HexToAddress(order.Maker.Hex()).Hex(),
		"signer":        common.HexToAddress(order.Signer.Hex()).Hex(),
		"taker":         common.HexToAddress(order.Taker.Hex()).Hex(),
		"tokenId":       bigIntString(order.TokenId),
		"makerAmount":   bigIntString(order.MakerAmount),
		"takerAmount":   bigIntString(order.TakerAmount),
		"expiration":    bigIntString(order.Expiration),
		"nonce":         bigIntString(order.Nonce),
		"feeRateBps":    bigIntString(order.FeeRateBps),
		"side":          sideStr,                               // 字符串 "BUY" 或 "SELL"
		"signatureType": int(bigIntInt64(order.SignatureType)), // 整数
		"signature":     signatureHex,
	}
	return map[string]interface{}{
		"order":     orderDict,
		"owner":     owner,
		"orderType": string(orderType),
		"postOnly":  postOnly,
		"deferExec": false,
	}
}

func signatureToHex(signature []byte) string {
	if signature == nil {
		return ""
	}

	sigStr := string(signature)
	if strings.HasPrefix(sigStr, "0x") {
		return sigStr
	}

	decoded, err := base64.StdEncoding.DecodeString(sigStr)
	if err == nil {
		return "0x" + hex.EncodeToString(decoded)
	}

	return "0x" + hex.EncodeToString(signature)
}

func bigIntString(v *big.Int) string {
	if v == nil {
		return "0"
	}
	return v.String()
}

func bigIntInt64(v *big.Int) int64 {
	if v == nil {
		return 0
	}
	return v.Int64()
}

// IsTickSizeSmaller 检查tick size是否更小
func IsTickSizeSmaller(a, b TickSize) bool {
	aFloat, _ := strconv.ParseFloat(string(a), 64)
	bFloat, _ := strconv.ParseFloat(string(b), 64)
	return aFloat < bFloat
}

// PriceValid 检查价格是否有效
func PriceValid(price float64, tickSize TickSize) bool {
	tickSizeFloat, _ := strconv.ParseFloat(string(tickSize), 64)
	return price >= tickSizeFloat && price <= 1.0-tickSizeFloat
}

// 辅助函数
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}
