package psbt_sdk

import "github.com/btcsuite/btcd/txscript"

type Input struct {
	OutTxId string `json:"out_tx_id"`
	OutIndex uint32 `json:"out_index"`
}

type UtxoType int

const (
	NonWitness UtxoType = 1
	Witness    UtxoType = 2
)

type InputUtxo struct {
	UtxoType            UtxoType             `json:"utxo_type"`
	SighashType         txscript.SigHashType `json:"sighash_type"`
	NonWitnessUtxo      string               `json:"non_witness_utxo"`       //
	WitnessUtxoPkScript string               `json:"witness_utxo_pk_script"` //
	WitnessUtxoAmount   uint64               `json:"witness_utxo_amount"`    //
	Index               int                  `json:"index"`
}

type InputSigner struct {
	UtxoType            UtxoType `json:"utxo_type"`
	SighashType         txscript.SigHashType `json:"sighash_type"`
	//Sig   string `json:"sig"`
	Pri  string `json:"pri"`
	Pub   string `json:"pub"`
	Index int    `json:"index"`
}