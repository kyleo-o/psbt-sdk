package psbt_sdk

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/psbt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

type PsbtBuilder struct {
	NetParams   *chaincfg.Params
	PsbtUpdater *psbt.Updater
}

// Create new psbt builder
func CreatePsbtBuilder(netParams *chaincfg.Params, ins []Input, outs []Output) (*PsbtBuilder, error) {
	var (
		txOuts     []*wire.TxOut    = make([]*wire.TxOut, 0)
		txIns      []*wire.OutPoint = make([]*wire.OutPoint, 0)
		nSequences []uint32         = make([]uint32, 0)
	)
	for _, in := range ins {
		txHash, err := chainhash.NewHashFromStr(in.OutTxId)
		if err != nil {
			return nil, err
		}
		prevOut := wire.NewOutPoint(txHash, in.OutIndex)
		txIns = append(txIns, prevOut)
		nSequences = append(nSequences, wire.MaxTxInSequenceNum)
	}

	for _, out := range outs {
		var pkScript []byte
		if out.Script != "" {
			scriptByte, err := hex.DecodeString(out.Script)
			if err != nil {
				return nil, err
			}
			pkScript = scriptByte
		} else {
			address, err := btcutil.DecodeAddress(out.Address, netParams)
			if err != nil {
				return nil, err
			}

			pkScript, err = txscript.PayToAddrScript(address)
			if err != nil {
				return nil, err
			}
		}

		txOut := wire.NewTxOut(int64(out.Amount), pkScript)
		txOuts = append(txOuts, txOut)
	}

	cPsbt, err := psbt.New(txIns, txOuts, int32(2), uint32(0), nSequences)
	if err != nil {
		return nil, err
	}
	psbtBuilder := &PsbtBuilder{NetParams: netParams}

	psbtBuilder.PsbtUpdater, err = psbt.NewUpdater(cPsbt)
	if err != nil {
		return nil, err
	}
	return psbtBuilder, nil
}

// add InputWitness without signed
func (s *PsbtBuilder) UpdateAndAddInputWitness(signIns []*InputSign) error {
	for _, v := range signIns {
		switch v.UtxoType {
		case Witness:
			witnessUtxoScriptHex, err := hex.DecodeString(v.PkScript)
			if err != nil {
				return err
			}
			txOut := wire.TxOut{Value: int64(v.Amount), PkScript: witnessUtxoScriptHex}
			err = s.PsbtUpdater.AddInWitnessUtxo(&txOut, v.Index)
			if err != nil {
				return err
			}
			err = s.PsbtUpdater.AddInSighashType(v.SighashType, v.Index)
			if err != nil {
				return err
			}
			break
		}
	}
	return nil
}

// add InputWitness with signed
func (s *PsbtBuilder) UpdateAndSignInput(signIns []*InputSign) error {
	for _, v := range signIns {
		privateKeyBytes, err := hex.DecodeString(v.PriHex)
		if err != nil {
			return err
		}
		privateKey, _ := btcec.PrivKeyFromBytes(privateKeyBytes)
		sigScript := []byte{}
		switch v.UtxoType {
		case NonWitness:
			tx := wire.NewMsgTx(2)
			nonWitnessUtxoHex, err := hex.DecodeString(v.OutRaw)
			if err != nil {
				return err
			}
			err = tx.Deserialize(bytes.NewReader(nonWitnessUtxoHex))
			if err != nil {
				return err
			}
			err = s.PsbtUpdater.AddInNonWitnessUtxo(tx, v.Index)
			if err != nil {
				return err
			}
			err = s.PsbtUpdater.AddInSighashType(v.SighashType, v.Index)
			if err != nil {
				return err
			}
			sigScript, err = txscript.RawTxInSignature(s.PsbtUpdater.Upsbt.UnsignedTx, v.Index, s.PsbtUpdater.Upsbt.Inputs[v.Index].NonWitnessUtxo.TxOut[s.PsbtUpdater.Upsbt.UnsignedTx.TxIn[v.Index].PreviousOutPoint.Index].PkScript, v.SighashType, privateKey)
			if err != nil {
				return err
			}
			break
		case Witness:
			witnessUtxoScriptHex, err := hex.DecodeString(v.PkScript)
			if err != nil {
				return err
			}
			txOut := wire.TxOut{Value: int64(v.Amount), PkScript: witnessUtxoScriptHex}
			err = s.PsbtUpdater.AddInWitnessUtxo(&txOut, v.Index)
			if err != nil {
				return err
			}
			err = s.PsbtUpdater.AddInSighashType(v.SighashType, v.Index)
			if err != nil {
				return err
			}
			prevOutputFetcher := NewPrevOutputFetcher(s.PsbtUpdater.Upsbt.Inputs[v.Index].WitnessUtxo.PkScript, s.PsbtUpdater.Upsbt.Inputs[v.Index].WitnessUtxo.Value)
			sigHashes := txscript.NewTxSigHashes(s.PsbtUpdater.Upsbt.UnsignedTx, prevOutputFetcher)
			sigScript, err = txscript.RawTxInWitnessSignature(s.PsbtUpdater.Upsbt.UnsignedTx, sigHashes,
				v.Index, s.PsbtUpdater.Upsbt.Inputs[v.Index].WitnessUtxo.Value,
				s.PsbtUpdater.Upsbt.Inputs[v.Index].WitnessUtxo.PkScript,
				v.SighashType, privateKey)
			if err != nil {
				return err
			}
			break
		}

		publicKey := hex.EncodeToString(privateKey.PubKey().SerializeCompressed())
		pubByte, err := hex.DecodeString(publicKey)
		if err != nil {
			return err
		}
		res, err := s.PsbtUpdater.Sign(v.Index, sigScript, pubByte, nil, nil)
		if err != nil || res != 0 {
			return errors.New(fmt.Sprintf("Index-[%d] %s, SignOutcome:%d", v.Index, err, res))
		}
		_, err = psbt.MaybeFinalize(s.PsbtUpdater.Upsbt, v.Index)
		if err != nil {
			return errors.New(fmt.Sprintf("Index-[%d] %s", v.Index, err))
		}
	}
	return nil
}

func (s *PsbtBuilder) UpdateAndSignInputNoFinalize(signIns []*InputSign) error {
	for _, v := range signIns {
		privateKeyBytes, err := hex.DecodeString(v.PriHex)
		if err != nil {
			return err
		}
		privateKey, _ := btcec.PrivKeyFromBytes(privateKeyBytes)
		sigScript := []byte{}
		switch v.UtxoType {
		case NonWitness:
			tx := wire.NewMsgTx(2)
			nonWitnessUtxoHex, err := hex.DecodeString(v.OutRaw)
			if err != nil {
				return err
			}
			err = tx.Deserialize(bytes.NewReader(nonWitnessUtxoHex))
			if err != nil {
				return err
			}
			err = s.PsbtUpdater.AddInNonWitnessUtxo(tx, v.Index)
			if err != nil {
				return err
			}
			err = s.PsbtUpdater.AddInSighashType(v.SighashType, v.Index)
			if err != nil {
				return err
			}
			sigScript, err = txscript.RawTxInSignature(s.PsbtUpdater.Upsbt.UnsignedTx, v.Index, s.PsbtUpdater.Upsbt.Inputs[v.Index].NonWitnessUtxo.TxOut[s.PsbtUpdater.Upsbt.UnsignedTx.TxIn[v.Index].PreviousOutPoint.Index].PkScript, v.SighashType, privateKey)
			if err != nil {
				return err
			}
			break
		case Witness:
			witnessUtxoScriptHex, err := hex.DecodeString(
				v.PkScript)
			if err != nil {
				return err
			}
			txOut := wire.TxOut{Value: int64(v.Amount), PkScript: witnessUtxoScriptHex}
			err = s.PsbtUpdater.AddInWitnessUtxo(&txOut, v.Index)
			if err != nil {
				return err
			}
			err = s.PsbtUpdater.AddInSighashType(v.SighashType, v.Index)
			if err != nil {
				return err
			}
			prevOutputFetcher := NewPrevOutputFetcher(s.PsbtUpdater.Upsbt.Inputs[v.Index].WitnessUtxo.PkScript, s.PsbtUpdater.Upsbt.Inputs[v.Index].WitnessUtxo.Value)
			sigHashes := txscript.NewTxSigHashes(s.PsbtUpdater.Upsbt.UnsignedTx, prevOutputFetcher)
			sigScript, err = txscript.RawTxInWitnessSignature(s.PsbtUpdater.Upsbt.UnsignedTx, sigHashes,
				v.Index, s.PsbtUpdater.Upsbt.Inputs[v.Index].WitnessUtxo.Value,
				s.PsbtUpdater.Upsbt.Inputs[v.Index].WitnessUtxo.PkScript,
				v.SighashType, privateKey)
			if err != nil {
				return err
			}
			break
		}

		publicKey := hex.EncodeToString(privateKey.PubKey().SerializeCompressed())
		pubByte, err := hex.DecodeString(publicKey)
		if err != nil {
			return err
		}
		res, err := s.PsbtUpdater.Sign(v.Index, sigScript, pubByte, nil, nil)
		if err != nil || res != 0 {
			return err
		}
		_, err = psbt.MaybeFinalize(s.PsbtUpdater.Upsbt, v.Index)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *PsbtBuilder) UpdateAndMultiSignInput(signIns []*InputSign) error {
	for _, v := range signIns {
		var (
			multiSigScriptByte []byte
			//preSigScriptBytes  []byte
			err error
		)
		//if v.PreSigScript != "" {
		//	//preSigScriptBytes, err = hex.DecodeString(v.PreSigScript)
		//	//if err != nil {
		//	//	return err
		//	//}
		//}
		if v.MultiSigScript != "" {
			multiSigScriptByte, err = hex.DecodeString(v.MultiSigScript)
			if err != nil {
				return err
			}
		}

		privateKeyBytes, err := hex.DecodeString(v.PriHex)
		if err != nil {
			return err
		}
		privateKey, _ := btcec.PrivKeyFromBytes(privateKeyBytes)

		sigScript := []byte{}
		witnessUtxoScriptHex, err := hex.DecodeString(
			v.PkScript)
		if err != nil {
			return err
		}
		txOut := wire.TxOut{Value: int64(v.Amount), PkScript: witnessUtxoScriptHex}
		err = s.PsbtUpdater.AddInWitnessUtxo(&txOut, v.Index)
		if err != nil {
			return err
		}
		err = s.PsbtUpdater.AddInSighashType(v.SighashType, v.Index)
		if err != nil {
			return err
		}

		prevOutputFetcher := NewPrevOutputFetcher(s.PsbtUpdater.Upsbt.Inputs[v.Index].WitnessUtxo.PkScript, s.PsbtUpdater.Upsbt.Inputs[v.Index].WitnessUtxo.Value)
		sigHashes := txscript.NewTxSigHashes(s.PsbtUpdater.Upsbt.UnsignedTx, prevOutputFetcher)
		sigScript, err = txscript.RawTxInWitnessSignature(s.PsbtUpdater.Upsbt.UnsignedTx, sigHashes,
			v.Index, s.PsbtUpdater.Upsbt.Inputs[v.Index].WitnessUtxo.Value,
			multiSigScriptByte,
			v.SighashType, privateKey)
		if err != nil {
			return err
		}

		publicKey := hex.EncodeToString(privateKey.PubKey().SerializeCompressed())
		pubByte, err := hex.DecodeString(publicKey)
		if err != nil {
			return err
		}
		//fmt.Printf("index:%d\n, pri:%s\n, pub:%s\n, sigScript: %s\n", v.Index, v.PriHex, publicKey, hex.EncodeToString(sigScript))
		res, err := s.PsbtUpdater.Sign(v.Index, sigScript, pubByte, nil, multiSigScriptByte)
		if err != nil || res != 0 {
			return err
		}

	}
	return nil
}

func (s *PsbtBuilder) AddSinInStruct(sigIn *SigIn) error {
	return s.AddSigIn(sigIn.WitnessUtxo, sigIn.SighashType, sigIn.FinalScriptWitness, sigIn.Index)
}

func (s *PsbtBuilder) AddSigIn(witnessUtxo *wire.TxOut, sighashType txscript.SigHashType, finalScriptWitness []byte, index int) error {
	s.PsbtUpdater.Upsbt.Inputs[index].SighashType = sighashType
	s.PsbtUpdater.Upsbt.Inputs[index].WitnessUtxo = witnessUtxo
	s.PsbtUpdater.Upsbt.Inputs[index].FinalScriptWitness = finalScriptWitness
	if err := s.PsbtUpdater.Upsbt.SanityCheck(); err != nil {
		return err
	}
	return nil
}

func (s *PsbtBuilder) AddMultiSigIn(witnessUtxo *wire.TxOut, sighashType txscript.SigHashType, ScriptWitness []byte, index int) error {
	s.PsbtUpdater.Upsbt.Inputs[index].SighashType = sighashType
	s.PsbtUpdater.Upsbt.Inputs[index].WitnessUtxo = witnessUtxo
	s.PsbtUpdater.Upsbt.Inputs[index].WitnessScript = ScriptWitness
	if err := s.PsbtUpdater.Upsbt.SanityCheck(); err != nil {
		return err
	}
	return nil
}

func (s *PsbtBuilder) ToString() (string, error) {
	var b bytes.Buffer
	err := s.PsbtUpdater.Upsbt.Serialize(&b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b.Bytes()), nil
}

func NewPsbtBuilder(netParams *chaincfg.Params, psbtHex string) (*PsbtBuilder, error) {
	psbtBuilder := &PsbtBuilder{NetParams: netParams}

	b, err := hex.DecodeString(psbtHex)
	if err != nil {
		return nil, err
	}
	p, err := psbt.NewFromRawBytes(bytes.NewReader(b), false)
	if err != nil {
		return nil, err
	}
	psbtBuilder.PsbtUpdater, err = psbt.NewUpdater(p)
	if err != nil {
		return nil, err
	}
	return psbtBuilder, nil
}

func (s *PsbtBuilder) GetInputs() []*wire.TxIn {
	return s.PsbtUpdater.Upsbt.UnsignedTx.TxIn
}

func (s *PsbtBuilder) GetOutputs() []*wire.TxOut {
	return s.PsbtUpdater.Upsbt.UnsignedTx.TxOut
}

func (s *PsbtBuilder) AddInput(in Input, signIn *InputSign) error {
	txHash, err := chainhash.NewHashFromStr(in.OutTxId)
	if err != nil {
		return err
	}
	s.PsbtUpdater.Upsbt.UnsignedTx.AddTxIn(&wire.TxIn{
		PreviousOutPoint: *wire.NewOutPoint(txHash, in.OutIndex),
		Sequence:         wire.MaxTxInSequenceNum,
	})
	s.PsbtUpdater.Upsbt.Inputs = append(s.PsbtUpdater.Upsbt.Inputs, psbt.PInput{})

	privateKeyBytes, err := hex.DecodeString(signIn.PriHex)
	if err != nil {
		return err
	}
	privateKey, _ := btcec.PrivKeyFromBytes(privateKeyBytes)
	sigScript := []byte{}
	var witnessScript wire.TxWitness
	switch signIn.UtxoType {
	case NonWitness:
		tx := wire.NewMsgTx(2)
		nonWitnessUtxoHex, err := hex.DecodeString(signIn.OutRaw)
		if err != nil {
			return err
		}
		err = tx.Deserialize(bytes.NewReader(nonWitnessUtxoHex))
		if err != nil {
			return err
		}
		//fmt.Printf("nonWitnessUtxoHe-tx: %s\n", tx.TxHash().String())
		err = s.PsbtUpdater.AddInNonWitnessUtxo(tx, signIn.Index)
		if err != nil {
			return err
		}
		err = s.PsbtUpdater.AddInSighashType(signIn.SighashType, signIn.Index)
		if err != nil {
			return err
		}
		sigScript, err = txscript.RawTxInSignature(s.PsbtUpdater.Upsbt.UnsignedTx, signIn.Index,
			s.PsbtUpdater.Upsbt.Inputs[signIn.Index].NonWitnessUtxo.TxOut[in.OutIndex].PkScript, signIn.SighashType, privateKey)

		if err != nil {
			return err
		}

		break
	case Witness:
		witnessUtxoScriptHex, err := hex.DecodeString(
			signIn.PkScript)
		if err != nil {
			return err
		}
		txout := wire.TxOut{Value: int64(signIn.Amount), PkScript: witnessUtxoScriptHex}
		err = s.PsbtUpdater.AddInWitnessUtxo(&txout, signIn.Index)
		if err != nil {
			return err
		}
		err = s.PsbtUpdater.AddInSighashType(signIn.SighashType, signIn.Index)
		if err != nil {
			return err
		}
		prevOutputFetcher := NewPrevOutputFetcher(s.PsbtUpdater.Upsbt.Inputs[signIn.Index].WitnessUtxo.PkScript, s.PsbtUpdater.Upsbt.Inputs[signIn.Index].WitnessUtxo.Value)
		sigHashes := txscript.NewTxSigHashes(s.PsbtUpdater.Upsbt.UnsignedTx, prevOutputFetcher)
		sigScript, err = txscript.RawTxInWitnessSignature(s.PsbtUpdater.Upsbt.UnsignedTx, sigHashes, signIn.Index, s.PsbtUpdater.Upsbt.Inputs[signIn.Index].WitnessUtxo.Value, s.PsbtUpdater.Upsbt.Inputs[signIn.Index].WitnessUtxo.PkScript, signIn.SighashType, privateKey)
		if err != nil {
			return err
		}
		break
	}
	_ = witnessScript
	publicKey := hex.EncodeToString(privateKey.PubKey().SerializeCompressed())
	pubByte, err := hex.DecodeString(publicKey)
	if err != nil {
		return err
	}
	res, err := s.PsbtUpdater.Sign(signIn.Index, sigScript, pubByte, nil, nil)
	if err != nil || res != 0 {
		return err
	}
	_, err = psbt.MaybeFinalize(s.PsbtUpdater.Upsbt, signIn.Index)
	if err != nil {
		return err
	}
	return nil
}

func (s *PsbtBuilder) AddOutput(outs []Output) error {
	txOuts := make([]*wire.TxOut, 0)
	for _, out := range outs {
		var pkScript []byte
		if out.Script != "" {
			scriptByte, err := hex.DecodeString(out.Script)
			if err != nil {
				return err
			}
			pkScript = scriptByte
		} else {
			address, err := btcutil.DecodeAddress(out.Address, s.NetParams)
			if err != nil {
				return err
			}

			pkScript, err = txscript.PayToAddrScript(address)
			if err != nil {
				return err
			}
		}

		txOut := wire.NewTxOut(int64(out.Amount), pkScript)
		txOuts = append(txOuts, txOut)
	}

	for _, out := range txOuts {
		s.PsbtUpdater.Upsbt.UnsignedTx.AddTxOut(out)
	}

	s.PsbtUpdater.Upsbt.Outputs = make([]psbt.POutput, len(s.PsbtUpdater.Upsbt.UnsignedTx.TxOut))
	return nil
}

func (s *PsbtBuilder) IsComplete() bool {
	return s.PsbtUpdater.Upsbt.IsComplete()
}

func (s *PsbtBuilder) CalculateFee(feeRate int64, extraSize int64) (int64, error) {
	txHex, err := s.ExtractPsbtTransaction()
	if err != nil {
		return 0, err
	}
	txByte, err := hex.DecodeString(txHex)
	if err != nil {
		return 0, err
	}
	fee := (int64(len(txByte)) + extraSize) * feeRate
	return fee, nil
}

func (s *PsbtBuilder) ExtractPsbtTransaction() (string, error) {
	if !s.IsComplete() {
		for i := range s.PsbtUpdater.Upsbt.UnsignedTx.TxIn {
			success, err := psbt.MaybeFinalize(s.PsbtUpdater.Upsbt, i)
			if err != nil || !success {
				return "", err
			}
		}

		err := psbt.MaybeFinalizeAll(s.PsbtUpdater.Upsbt)
		if err != nil {
			return "", err
		}
	}

	tx, err := psbt.Extract(s.PsbtUpdater.Upsbt)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	err = tx.Serialize(&b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b.Bytes()), nil
}

type PrevOutputFetcher struct {
	pkScript []byte
	value    int64
}

func NewPrevOutputFetcher(pkScript []byte, value int64) *PrevOutputFetcher {
	return &PrevOutputFetcher{
		pkScript,
		value,
	}
}

func (d *PrevOutputFetcher) FetchPrevOutput(wire.OutPoint) *wire.TxOut {
	return &wire.TxOut{
		Value:    d.value,
		PkScript: d.pkScript,
	}
}
