# PSBT SDK

[中文文档](README_CN.md) | English

A comprehensive Go SDK for creating, managing, and signing Bitcoin Partially Signed Bitcoin Transactions (PSBTs). This SDK provides a simple and powerful interface for building complex Bitcoin transactions with support for various UTXO types including Legacy, SegWit, and Taproot.

## Installation

```bash
go get github.com/kyleo-o/psbt-sdk
```

## Quick Start

### Basic PSBT Creation

```go
package main

import (
    "log"
    "github.com/btcsuite/btcd/chaincfg"
    "github.com/kyleo-o/psbt-sdk"
)

func main() {
    // Set network parameters
    netParams := &chaincfg.MainNetParams
    
    // Define inputs (UTXOs to spend)
    inputs := []psbt_sdk.Input{
        {
            OutTxId:  "your_tx_id_here",
            OutIndex: 0,
        },
    }
    
    // Define outputs (where to send Bitcoin)
    outputs := []psbt_sdk.Output{
        {
            Address: "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa", // Example address
            Amount:  100000, // Amount in satoshis
        },
    }
    
    // Create PSBT builder
    builder, err := psbt_sdk.CreatePsbtBuilder(netParams, inputs, outputs)
    if err != nil {
        log.Fatal(err)
    }
    
    // Get PSBT as hex string
    psbtHex, err := builder.ToString()
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("PSBT: %s", psbtHex)
}
```

### Signing a PSBT

```go
// Define signing information
signInputs := []*psbt_sdk.InputSign{
    {
        UtxoType:    psbt_sdk.NonWitness, // or psbt_sdk.Witness, psbt_sdk.Taproot
        Index:       0,
        Amount:      100000,
        SighashType: txscript.SigHashAll,
        PriHex:      "your_private_key_hex",
    },
}

// Sign the PSBT
err = builder.UpdateAndSignInput(signInputs)
if err != nil {
    log.Fatal(err)
}

// Extract final transaction
txHex, err := builder.ExtractPsbtTransaction()
if err != nil {
    log.Fatal(err)
}

log.Printf("Final Transaction: %s", txHex)
```

## API Reference

### Core Types

#### Input
```go
type Input struct {
    OutTxId  string `json:"out_tx_id"`  // Previous transaction ID
    OutIndex uint32 `json:"out_index"`  // Output index in previous transaction
}
```

#### Output
```go
type Output struct {
    Address string `json:"address"` // Bitcoin address
    Amount  uint64 `json:"amount"`  // Amount in satoshis
    Script  string `json:"script"`  // Optional: custom script
}
```

#### InputSign
```go
type InputSign struct {
    UtxoType            UtxoType             `json:"utxo_type"`
    Index               int                  `json:"index"`
    OutRaw              string               `json:"out_raw"`
    PkScript            string               `json:"pk_script"`
    RedeemScript        string               `json:"redeem_script"`
    ControlBlockWitness string               `json:"control_block_witness"`
    Amount              uint64               `json:"amount"`
    SighashType         txscript.SigHashType `json:"sighash_type"`
    PriHex              string               `json:"pri_hex"`
    MultiSigScript      string               `json:"multi_sig_script"`
    PreSigScript        string               `json:"pre_sig_script"`
}
```

### Main Functions

#### CreatePsbtBuilder
Creates a new PSBT builder with inputs and outputs.

```go
func CreatePsbtBuilder(netParams *chaincfg.Params, ins []Input, outs []Output) (*PsbtBuilder, error)
```

#### NewPsbtBuilder
Creates a PSBT builder from an existing PSBT hex string.

```go
func NewPsbtBuilder(netParams *chaincfg.Params, psbtHex string) (*PsbtBuilder, error)
```

### PsbtBuilder Methods

#### Signing Methods

- `UpdateAndSignInput(signIns []*InputSign) error` - Sign inputs (Legacy/SegWit)
- `UpdateAndSignTaprootInput(signIns []*InputSign) error` - Sign Taproot inputs
- `UpdateAndSignInputNoFinalize(signIns []*InputSign) error` - Sign without finalizing
- `UpdateAndMultiSignInput(signIns []*InputSign) error` - Multi-signature signing

#### Transaction Building

- `AddInput(in Input, signIn *InputSign) error` - Add input to transaction
- `AddOutput(outs []Output) error` - Add outputs to transaction
- `AddInputOnly(in Input) error` - Add input without signing info

#### Utility Methods

- `ToString() (string, error)` - Get PSBT as hex string
- `ExtractPsbtTransaction() (string, error)` - Extract final transaction
- `IsComplete() bool` - Check if PSBT is complete
- `CalculateFee(feeRate int64, extraSize int64) (int64, error)` - Calculate fees
- `CalTxSize() (int64, error)` - Calculate transaction size

## Examples

### Legacy Transaction

```go
// Create inputs and outputs
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

// Create and sign
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

### SegWit Transaction

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

### NestSegwit Transaction

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

### Taproot Transaction

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

## UTXO Types

The SDK supports three types of UTXOs:

- `NonWitness` (1) - Legacy P2PKH/P2SH transactions
- `Witness` (2) - SegWit P2WPKH/P2WSH transactions  
- `Taproot` (3) - Taproot P2TR transactions

## Network Support

- Bitcoin Mainnet
- Bitcoin Testnet
- Bitcoin Signet
- Bitcoin Regression Test

## Dependencies

- `github.com/btcsuite/btcd` - Bitcoin protocol implementation
- `github.com/btcsuite/btcd/btcutil` - Bitcoin utilities
- `github.com/btcsuite/btcd/btcutil/psbt` - PSBT implementation

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For questions and support, please open an issue on GitHub.
