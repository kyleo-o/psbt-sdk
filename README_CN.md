# PSBT SDK

English | [中文文档](README_CN.md)

一个功能全面的Go语言SDK，用于创建、管理和签名比特币部分签名比特币交易（PSBT）。该SDK提供了简单而强大的接口，用于构建复杂的比特币交易，支持各种UTXO类型，包括Legacy、SegWit和Taproot。

## 安装

```bash
go get github.com/kyleo-o/psbt-sdk
```

## 快速开始

### 基础PSBT创建

```go
package main

import (
    "log"
    "github.com/btcsuite/btcd/chaincfg"
    "github.com/kyleo-o/psbt-sdk"
)

func main() {
    // 设置网络参数
    netParams := &chaincfg.MainNetParams
    
    // 定义输入（要花费的UTXO）
    inputs := []psbt_sdk.Input{
        {
            OutTxId:  "your_tx_id_here",
            OutIndex: 0,
        },
    }
    
    // 定义输出（发送比特币的地址）
    outputs := []psbt_sdk.Output{
        {
            Address: "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa", // 示例地址
            Amount:  100000, // 金额（聪）
        },
    }
    
    // 创建PSBT构建器
    builder, err := psbt_sdk.CreatePsbtBuilder(netParams, inputs, outputs)
    if err != nil {
        log.Fatal(err)
    }
    
    // 获取PSBT十六进制字符串
    psbtHex, err := builder.ToString()
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("PSBT: %s", psbtHex)
}
```

### 签名PSBT

```go
// 定义签名信息
signInputs := []*psbt_sdk.InputSign{
    {
        UtxoType:    psbt_sdk.NonWitness, // 或 psbt_sdk.Witness, psbt_sdk.Taproot
        Index:       0,
        Amount:      100000,
        SighashType: txscript.SigHashAll,
        PriHex:      "your_private_key_hex",
    },
}

// 签名PSBT
err = builder.UpdateAndSignInput(signInputs)
if err != nil {
    log.Fatal(err)
}

// 提取最终交易
txHex, err := builder.ExtractPsbtTransaction()
if err != nil {
    log.Fatal(err)
}

log.Printf("最终交易: %s", txHex)
```

## API参考

### 核心类型

#### Input（输入）
```go
type Input struct {
    OutTxId  string `json:"out_tx_id"`  // 前一个交易ID
    OutIndex uint32 `json:"out_index"`  // 前一个交易中的输出索引
}
```

#### Output（输出）
```go
type Output struct {
    Address string `json:"address"` // 比特币地址
    Amount  uint64 `json:"amount"`  // 金额（聪）
    Script  string `json:"script"`  // 可选：自定义脚本
}
```

#### InputSign（输入签名）
```go
type InputSign struct {
    UtxoType            UtxoType             `json:"utxo_type"`            // UTXO类型
    Index               int                  `json:"index"`                // 索引
    OutRaw              string               `json:"out_raw"`              // 原始输出
    PkScript            string               `json:"pk_script"`            // 公钥脚本
    RedeemScript        string               `json:"redeem_script"`         // 赎回脚本
    ControlBlockWitness string               `json:"control_block_witness"`  // 控制块见证
    Amount              uint64               `json:"amount"`               // 金额
    SighashType         txscript.SigHashType `json:"sighash_type"`          // 签名哈希类型
    PriHex              string               `json:"pri_hex"`               // 私钥十六进制
    MultiSigScript      string               `json:"multi_sig_script"`       // 多重签名脚本
    PreSigScript        string               `json:"pre_sig_script"`         // 预签名脚本
}
```

### 主要函数

#### CreatePsbtBuilder
使用输入和输出创建新的PSBT构建器。

```go
func CreatePsbtBuilder(netParams *chaincfg.Params, ins []Input, outs []Output) (*PsbtBuilder, error)
```

#### NewPsbtBuilder
从现有的PSBT十六进制字符串创建PSBT构建器。

```go
func NewPsbtBuilder(netParams *chaincfg.Params, psbtHex string) (*PsbtBuilder, error)
```

### PsbtBuilder方法

#### 签名方法

- `UpdateAndSignInput(signIns []*InputSign) error` - 签名输入（Legacy/SegWit）
- `UpdateAndSignTaprootInput(signIns []*InputSign) error` - 签名Taproot输入
- `UpdateAndSignInputNoFinalize(signIns []*InputSign) error` - 签名但不完成
- `UpdateAndMultiSignInput(signIns []*InputSign) error` - 多重签名

#### 交易构建

- `AddInput(in Input, signIn *InputSign) error` - 向交易添加输入
- `AddOutput(outs []Output) error` - 向交易添加输出
- `AddInputOnly(in Input) error` - 仅添加输入（无签名信息）

#### 工具方法

- `ToString() (string, error)` - 获取PSBT十六进制字符串
- `ExtractPsbtTransaction() (string, error)` - 提取最终交易
- `IsComplete() bool` - 检查PSBT是否完成
- `CalculateFee(feeRate int64, extraSize int64) (int64, error)` - 计算手续费
- `CalTxSize() (int64, error)` - 计算交易大小

## 示例

### Legacy交易

```go
// 创建输入和输出
inputs := []psbt_sdk.Input{
    {
        OutTxId:  "previous_tx_id",
        OutIndex: 0,
    },
}

outputs := []psbt_sdk.Output{
    {
        Address: "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
        Amount:  50000,
    },
}

// 创建并签名
builder, _ := psbt_sdk.CreatePsbtBuilder(netParams, inputs, outputs)
signInputs := []*psbt_sdk.InputSign{
    {
        UtxoType:    psbt_sdk.NonWitness,
        Index:       0,
        Amount:      100000,
        OutRaw:      "previous_tx_raw",
        SighashType: txscript.SigHashAll,
        PriHex:      "private_key_hex",
    },
}

builder.UpdateAndSignInput(signInputs)
txHex, _ := builder.ExtractPsbtTransaction()
```

### SegWit交易

```go
signInputs := []*psbt_sdk.InputSign{
    {
        UtxoType:    psbt_sdk.Witness,
        Index:       0,
        Amount:      100000,
        SighashType: txscript.SigHashAll,
        PriHex:      "private_key_hex",
    },
}

builder.UpdateAndSignInput(signInputs)
```


### NestSegwit交易

```go
signInputs := []*psbt_sdk.InputSign{
    {
        UtxoType:    psbt_sdk.Witness,
        Index:       0,
        PkScript:     "pk_script",
        RedeemScript: "redeem_script",
        Amount:      100000,
        SighashType: txscript.SigHashAll,
        PriHex:      "private_key_hex",
    },
}

builder.UpdateAndSignInput(signInputs)
```

### Taproot交易

```go
signInputs := []*psbt_sdk.InputSign{
    {
        UtxoType:    psbt_sdk.Taproot,
        Index:       0,
        PkScript:     "pk_script",
        Amount:      100000,
        SighashType: txscript.SigHashDefault,
        PriHex:      "private_key_hex",
    },
}

builder.UpdateAndSignTaprootInput(signInputs)
```

## UTXO类型

SDK支持三种UTXO类型：

- `NonWitness` (1) - Legacy P2PKH/P2SH交易
- `Witness` (2) - SegWit P2WPKH/P2WSH交易  
- `Taproot` (3) - Taproot P2TR交易

## 网络支持

- 比特币主网
- 比特币测试网
- 比特币Signet
- 比特币回归测试

## 依赖

- `github.com/btcsuite/btcd` - 比特币协议实现
- `github.com/btcsuite/btcd/btcutil` - 比特币工具
- `github.com/btcsuite/btcd/btcutil/psbt` - PSBT实现

## 许可证

本项目采用 MIT 许可证 - 查看 LICENSE 文件了解详情。

## 支持

如有问题和支持需求，请在 GitHub 上提交 issue。
