package psbt_sdk

import (
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	"log"
	"testing"
)

func TestPsbtBuilder_CreatePsbtTransaction(t *testing.T) {
	s := &PsbtBuilder{
		NetParams:   &chaincfg.SigNetParams,
	}
	ins := []Input{
		{
			OutTxId:  "",
			OutIndex: 0,
		},
		{
			OutTxId:  "",
			OutIndex: 0,
		},
	}

	outs := []Output{
		{
			Address: "",
			Amount:  0,
		},
		{
			Address: "",
			Amount:  0,
		},
	}

	if err := s.CreatePsbtTransaction(ins, outs); err != nil {
		log.Fatalf("CreatePsbtTransaction() error = %v,", err)
	}


	inUtxos := []InputUtxo{
		{
			UtxoType:          0,
			NonWitnessUtxo:    "",
			WitnessUtxo:       "",
			WitnessUtxoAmount: 0,
			Index:             0,
		},
	}

	if err := s.UpdatePsbtTransaction(inUtxos); err != nil {
		log.Fatalf("UpdatePsbtTransaction() error = %v,", err)
	}


	inSigners := []InputSigner{
		{
			Sig:   "",
			Pub:   "",
			Index: 0,
		},
	}
	if err := s.SignPsbtTransaction(inSigners); err != nil {
		log.Fatalf("SignPsbtTransaction() error = %v,", err)
	}

	raw, err := s.PsbtTransactionToRawString()
	if err != nil {
		log.Fatalf("PsbtTransactionToRawString() error = %v,", err)
	}

	fmt.Printf("Raw:%s\n", raw)
}