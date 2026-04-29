# pUSD Redeem Check

这个示例用于检查本次修复：Polygon 主网 Web3 collateral 是否已经从旧 USDC.e 切到 pUSD。

它不会发交易，只会创建 Web3 client 并断言 `client.USDCAddress` 等于 pUSD 地址：

`0xC011a7E12a19f7B1f670d46F03B03f3342E82DFB`

## 运行

```bash
./run.sh
```

可选检查当前钱包的 pUSD 余额：

```bash
PRIVATE_KEY=0x... RPC_URL=https://polygon-rpc.com CHECK_BALANCE=1 ./run.sh
```

## 预期结果

输出里应该看到：

```text
Configured collateral: 0xC011a7E12a19f7B1f670d46F03B03f3342E82DFB
Status: OK - redeem/split/merge collateral is pUSD
```

