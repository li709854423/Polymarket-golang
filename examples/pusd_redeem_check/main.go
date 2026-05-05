package main

import (
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/li709854423/Polymarket-golang/polymarket/web3"
)

const (
	expectedPUSDAddress = "0xC011a7E12a19f7B1f670d46F03B03f3342E82DFB"
	dummyPrivateKey     = "0x0000000000000000000000000000000000000000000000000000000000000001"
)

func main() {
	chainID := int64(137)
	if chainIDStr := os.Getenv("CHAIN_ID"); chainIDStr != "" {
		parsed, err := strconv.ParseInt(chainIDStr, 10, 64)
		if err != nil {
			log.Fatalf("invalid CHAIN_ID: %v", err)
		}
		chainID = parsed
	}

	rpcURL := os.Getenv("RPC_URL")
	privateKey := os.Getenv("PRIVATE_KEY")
	if privateKey == "" {
		privateKey = dummyPrivateKey
	}

	client, err := web3.NewPolymarketWeb3Client(
		privateKey,
		web3.SignatureTypeEOA,
		chainID,
		rpcURL,
	)
	if err != nil {
		log.Fatalf("failed to create Web3 client: %v", err)
	}

	expected := common.HexToAddress(expectedPUSDAddress)
	actual := client.USDCAddress

	fmt.Println("=== pUSD collateral check ===")
	fmt.Printf("Chain ID: %d\n", chainID)
	fmt.Printf("Wallet: %s\n", client.Address.Hex())
	fmt.Printf("Configured collateral: %s\n", actual.Hex())
	fmt.Printf("Expected pUSD:        %s\n", expected.Hex())

	if actual != expected {
		log.Fatalf("collateral mismatch: got %s, want %s", actual.Hex(), expected.Hex())
	}

	fmt.Println("Status: OK - redeem/split/merge collateral is pUSD")

	if os.Getenv("CHECK_BALANCE") == "1" {
		balance, err := client.GetUSDCBalance(common.Address{})
		if err != nil {
			log.Fatalf("failed to get pUSD balance: %v", err)
		}
		fmt.Printf("pUSD Balance: %s\n", formatBalance(balance))
	}
}

func formatBalance(balance *big.Float) string {
	value, _ := balance.Float64()
	return fmt.Sprintf("%.6f", value)
}
