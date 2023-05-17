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
		//{
		//	//0.0009
		//	OutTxId:  "70def774e8c57fdacf79cca04abace87b0181daa037ec2cccf82e530b0e65967",
		//	OutIndex: 0,
		//},
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
		//{
		//	Address: "moPiaBwnvbowi3YMJ1UmGTjDUEyk2ckV39",
		//	Amount:  50000,
		//},
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
		//{
		//	UtxoType:          NonWitness,
		//	SighashType:txscript.SigHashSingle,
		//	NonWitnessUtxo:    "020000000001011c8e7ebec562b931c4aa423636a124026176c989c26149d654adad7db832abbe0000000000feffffff0240420f00000000001600145662a3e128e04aef586b4cd9ce79cdd06fae97e77cba37c44d060000160014fea0e3baa61b4e167e4d0c5881cc5cb0c48e31e002473044022023161fd137c23c64f416336e62c749290fdfac3f4dfa3d36cf7abb3ea077d35102202469f9c0295ebd8278e2f763ae03bb2e41f40fd21db8c74a6adc6cdf576a0ad2012102525c63e14f5c4c87c69cde6800de1f4cd0367fe3b4590a5c8b3b76944ef9f45f272e0200",
		//	WitnessUtxoPkScript:       "",
		//	WitnessUtxoAmount: 1000000,
		//	Index:             1,
		//},
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
		log.Fatalf("PsbtTransactionToRawString() error = %v,", err)
	}

	fmt.Printf("Raw:%s\n", raw)
}