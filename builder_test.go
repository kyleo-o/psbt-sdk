package psbt_sdk

import (
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"log"
	"testing"
)

func TestPsbtBuilder_CreatePsbtTransaction(t *testing.T) {
	var (
		netParams = &chaincfg.SigNetParams

		builder1 *PsbtBuilder
		inputs   []Input
		outputs  []Output
		inSigns  []*InputSign

		builder2  *PsbtBuilder
		newInput  Input
		newInSign *InputSign

		err      error
		psbtRaw1 string = ""
		psbtRaw2 string = ""
		txRaw    string = ""
	)
	//make SigHashSingle psbt
	inputs = []Input{
		{
			//0.00100000
			OutTxId:  "4db9ef8a51c06267fc1def09f21c79bc9f5ab3d3ba618edcfa18b5dc13340140",
			OutIndex: 0,
		},
	}

	outputs = []Output{
		{
			Address: "mqNXHVJMZPJ64YjKsAp8hn912cXeqpeKwL",
			Amount:  120000,
		},
	}
	builder1, err = CreatePsbtBuilder(netParams, inputs, outputs)
	if err != nil {
		log.Fatalf("CreatePsbtBuilder() error = %v,", err)
	}

	inSigns = []*InputSign{
		{
			UtxoType:    NonWitness,
			SighashType: txscript.SigHashSingle | txscript.SigHashAnyOneCanPay,
			OutRaw:      "0200000000010170adcc6698827b45f392408ed3164684cd956dc86e0b862b5096d8cc68b8f6660000000000feffffff02a0860100000000001976a9146c190d953f764e6763323a25c73764b7f39d380588acd91d036f4d0600001600147d7c80c8ebabfa4792994764923b8038b8d859d60247304402204cf7d8a5912469f1a60995b1238ab5c1fb32db0657ca30b07e1ff9c998e0cf9b02200dbc30bbf961702ab99e2efb34c934633d88d738e385a46e678e2880dfc7cbce012103ae93aac2be99194904389d369fe0d3b518f1538ce1520089b41732a2d0e0a2a01cb10200",
			PkScript:    "",
			Amount:      100000,
			Index:       0,
			PriHex:      "cff0c69901d49a23c6ce617d5779110630ee5616c00435879bba3f94cdaa0256", //mqNXHVJMZPJ64YjKsAp8hn912cXeqpeKwL
		},
	}

	if err = builder1.UpdateAndSignInput(inSigns); err != nil {
		log.Fatalf("UpdateAndSignInput() error = %v,", err)
	}

	psbtRaw1, err = builder1.ToString()
	if err != nil {
		log.Fatalf("ExtractPsbtTransaction() error = %v,", err)
	}

	fmt.Printf("PsbtRaw1:%s\n", psbtRaw1)

	//add new input in SigHashSingle psbt
	builder2, err = NewPsbtBuilder(netParams, psbtRaw1)
	if err != nil {
		log.Fatalf("NewPsbtBuilder() error = %v,", err)
	}

	newInput = Input{
		//0.00100000
		OutTxId:  "93b3484083c31769c6a82b9c1cb66979463d86b3edea5044dede1ffdb74c3db2",
		OutIndex: 0,
	}

	newInSign = &InputSign{
		UtxoType:    Witness,
		SighashType: txscript.SigHashAll,
		OutRaw:      "",
		PkScript:    "00144759c0016649844c814833c5afd9a4c748f49992",
		Amount:      100000,
		Index:       1,
		PriHex:      "3ce02df5120ce197680f58cc8038937090212f87d6eb09652c365196d161ab32", //tb1qgavuqqtxfxzyeq2gx0z6lkdycay0fxvj08gqeh
	}

	err = builder2.AddInput(newInput, newInSign)
	if err != nil {
		log.Fatalf("AddInput() error = %v,", err)
	}

	psbtRaw2, err = builder1.ToString()
	if err != nil {
		log.Fatalf("ExtractPsbtTransaction() error = %v,", err)
	}

	txRaw, err = builder2.ExtractPsbtTransaction()
	if err != nil {
		log.Fatalf("ExtractPsbtTransaction() error = %v,", err)
	}

	fmt.Printf("PsbtRaw2:%s\n", psbtRaw2)
	fmt.Printf("FinalRaw2:%s\n", txRaw)
	//https://mempool.space/zh/signet/tx/71d5492dce16436cbda2ff09e4bc3d19c90c0ceae1ea037cd8b2a4f572b66dea
}
