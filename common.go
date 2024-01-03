package psbt_sdk

type UtxoType int

const (
	NonWitness UtxoType = 1
	Witness    UtxoType = 2
)

const (
	OccupiedTxId    string = "0000000000000000000000000000000000000000000000000000000000000000"
	OccupiedTxIndex uint32 = 0
)
