package psbt_sdk

import (
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"log"
	"testing"
)

func TestPsbtBuilder_CreatePsbtTransaction(t *testing.T) {
	s := &PsbtBuilder{
		NetParams:   &chaincfg.SigNetParams,
	}
	ins := []Input{
		{
			//‎‎0.00700000
			OutTxId:  "43442b57048e0e188e1346caf7435372ddb9ce33b4da870a18d9818b0e4542ea",
			OutIndex: 0,
		},
		{
			//0.01000000
			OutTxId:  "456ae529982ebfe0e26b90f306b49df0c5a8b40537f9c28d6962549c7cb8262a",
			OutIndex: 0,
		},
	}

	outs := []Output{
		{
			Address: "moPiaBwnvbowi3YMJ1UmGTjDUEyk2ckV39",
			Amount:  700000,
		},
		{
			Address: "tb1q2e328cfgup9w7krtfnvuu7wd6ph6a9l8cdwakd",
			Amount:  900000,
		},
	}

	if err := s.CreatePsbtTransaction(ins, outs); err != nil {
		log.Fatalf("CreatePsbtTransaction() error = %v,", err)
	}


	inUtxos := []InputUtxo{
		{
			UtxoType:          NonWitness,
			SighashType:txscript.SigHashSingle|txscript.SigHashAnyOneCanPay,
			NonWitnessUtxo:    "020000000202da1468a43ba1474d4e1f50c17954ceb140631078b16fbfba211e58149b218f000000006a47304402203464da615ac05a9cf107ca830d2845d89f0ddb70ccf20ebde54b3497d3cf17ff022025a2223ad93ff6ff8fef09c0d5bc9067d16cfb7597c309abeeb26bd9b8d3e78b832103a286c4321008385ba8df8a77dee96badaf1fc07b05b2622bb846ff78f4b910ebffffffff6759e6b030e582cfccc27e03aa1d18b087ceba4aa0cc79cfda7fc5e874f7de70000000006b483045022100c66c370da1d64fee467f5a6730deee69f1109662754ffb4307e90307793e955902203bc989966640cfce169fcce0d95775d555986e54a4fceca8e59463e5b81fa78f032103a286c4321008385ba8df8a77dee96badaf1fc07b05b2622bb846ff78f4b910ebffffffff0260ae0a00000000001976a9145662a3e128e04aef586b4cd9ce79cdd06fae97e788ac50c30000000000001976a9145662a3e128e04aef586b4cd9ce79cdd06fae97e788ac00000000",
			WitnessUtxoPkScript:       "",
			WitnessUtxoAmount: 700000,
			Index:             0,
		},
		{
			UtxoType:          Witness,
			SighashType:txscript.SigHashSingle,
			NonWitnessUtxo:    "",
			WitnessUtxoPkScript:       "00145662a3e128e04aef586b4cd9ce79cdd06fae97e7",
			WitnessUtxoAmount: 1000000,
			Index:             1,
		},
	}

	if err := s.UpdatePsbtTransaction(inUtxos); err != nil {
		log.Fatalf("UpdatePsbtTransaction() error = %v,", err)
	}

	inSigners1 := []InputSigner{
		{
			UtxoType:          NonWitness,
			SighashType: txscript.SigHashSingle|txscript.SigHashAnyOneCanPay,
			Pri:   "4f302977e29281f228a7a208a4707e489996824085af1b05ad14a9b4a34edb5d",
			Pub:   "03a286c4321008385ba8df8a77dee96badaf1fc07b05b2622bb846ff78f4b910eb",
			Index: 0,
		},
	}
	if err := s.SignPsbtTransaction(inSigners1); err != nil {
		log.Fatalf("SignPsbtTransaction() error = %v,", err)
	}

	inSigners2 := []InputSigner{
		{
			UtxoType:          Witness,
			SighashType:txscript.SigHashSingle,
			Pri:   "4f302977e29281f228a7a208a4707e489996824085af1b05ad14a9b4a34edb5d",
			Pub:   "03a286c4321008385ba8df8a77dee96badaf1fc07b05b2622bb846ff78f4b910eb",
			Index: 1,
		},
	}
	if err := s.SignPsbtTransaction(inSigners2); err != nil {
		log.Fatalf("SignPsbtTransaction() error = %v,", err)
	}


	raw, err := s.ExtractPsbtTransaction()
	if err != nil {
		log.Fatalf("ExtractPsbtTransaction() error = %v,", err)
	}

	fmt.Printf("Raw:%s\n", raw)
}


func TestPsbtBuilder_NewUpdaterFromPsbtTransaction(t *testing.T) {
	psbtRaw := "70736274ff01005e0200000001e9f3c43d438fd5d359ccf99334824b473c2ea97d8cbcaa2c157609703959f7480100000000ffffffff01e80300000000000022512064729335d854ed8b52c83f02d7695630167de3f0bda3d511dfe881dd4d4aa4ca00000000000100fd090102000000000101d2ec50d5ef95d1f0811d3403fe84a6260472b51cc30a7e7158c794918d3cbcf20000000000feffffff03a97c1b0000000000160014abf3b63c1cb79ee3009989475d147273c013ba2a701700000000000022512064729335d854ed8b52c83f02d7695630167de3f0bda3d511dfe881dd4d4aa4cae803000000000000160014900ac7e578ced02be6d3db072328a76e9a6d3ee10247304402202c751e6be6b033e003b136968cc7c325f846412fff54494c8abe0176e0b05fc80220371419eeecf71815f6e51bb8400548facdb38ca85c13ec689e67124458bf3a910121021d7e2a3ce13a374facccc4385ecbb9b25f7b590d0ebd7bad06371e3710e575643a21250001012b701700000000000022512064729335d854ed8b52c83f02d7695630167de3f0bda3d511dfe881dd4d4aa4ca01084301418e5f4cabc6e3860c3e223abf79ba7793301a4e394d436d019c37ab44c9d1b6a3ca5b26974030926d830fb72e8121083d41a52495cc34a76e3e0a059cbdfde96a830000"
	s := &PsbtBuilder{
		NetParams:   &chaincfg.TestNet3Params,
	}
	if err := s.NewUpdaterFromPsbtTransaction(psbtRaw); err != nil {
		log.Fatalf("NewUpdaterFromPsbtTransaction() error = %v,", err)
	}
	outs := []Output{
		{
			Address: "tb1q2e328cfgup9w7krtfnvuu7wd6ph6a9l8cdwakd",
			Amount:  700000,
		},
	}
	if err := s.UpdaterAddOutputs(outs); err != nil {
		log.Fatalf("UpdaterAddOutputs() error = %v,", err)
	}


}