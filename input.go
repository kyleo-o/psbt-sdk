package psbt_sdk

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
	UtxoType          UtxoType `json:"utxo_type"`
	NonWitnessUtxo    string   `json:"non_witness_utxo"`    //
	WitnessUtxo       string   `json:"witness_utxo"`        //
	WitnessUtxoAmount uint64   `json:"witness_utxo_amount"` //
	Index             int      `json:"index"`
}

type InputSigner struct {
	Sig   string `json:"sig"`
	Pub   string `json:"pub"`
	Index int    `json:"index"`
}