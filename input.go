package psbt_sdk

import (
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

type Input struct {
	OutTxId  string `json:"out_tx_id"`
	OutIndex uint32 `json:"out_index"`
}

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

type SigIn struct {
	WitnessUtxo        *wire.TxOut          `json:"witnessUtxo"`
	SighashType        txscript.SigHashType `json:"sighashType"`
	FinalScriptWitness []byte               `json:"finalScriptWitness"`
	Index              int                  `json:"index"`
}
