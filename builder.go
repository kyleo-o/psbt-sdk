package psbt_sdk

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
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
			break
		case Witness, Taproot:
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

func (s *PsbtBuilder) UpdateAndSignTaprootInput(signIns []*InputSign) error {
	var witnessScript []byte
	_ = witnessScript

	prevOutputFetcher := txscript.NewMultiPrevOutFetcher(nil)
	for i, txIn := range s.PsbtUpdater.Upsbt.UnsignedTx.TxIn {
		outPoint := txIn.PreviousOutPoint
		txOut := s.PsbtUpdater.Upsbt.Inputs[i].WitnessUtxo
		if txOut == nil {
			txOut = s.PsbtUpdater.Upsbt.Inputs[i].NonWitnessUtxo.TxOut[outPoint.Index]
		}
		prevOutputFetcher.AddPrevOut(outPoint, txOut)
	}
	for _, v := range signIns {
		pkScript, err := hex.DecodeString(v.PkScript)
		if err != nil {
			return err
		}
		outPoint := s.PsbtUpdater.Upsbt.UnsignedTx.TxIn[v.Index].PreviousOutPoint
		txOut := wire.TxOut{Value: int64(v.Amount), PkScript: pkScript}
		err = s.PsbtUpdater.AddInWitnessUtxo(&txOut, v.Index)
		if err != nil {
			return err
		}
		err = s.PsbtUpdater.AddInSighashType(v.SighashType, v.Index)
		if err != nil {
			return err
		}
		prevOutputFetcher.AddPrevOut(outPoint, &txOut)
	}

	for _, v := range signIns {
		var (
			taprootKeySpendSig []byte
			err                error
		)
		//fmt.Printf("UpdateAndSignInput - signIn: %+v\n", v)
		privateKeyBytes, err := hex.DecodeString(v.PriHex)
		if err != nil {
			return err
		}
		privateKey, _ := btcec.PrivKeyFromBytes(privateKeyBytes)
		switch v.UtxoType {
		case Taproot:
			pkScript, err := hex.DecodeString(v.PkScript)
			if err != nil {
				return err
			}

			outPoint := s.PsbtUpdater.Upsbt.UnsignedTx.TxIn[v.Index].PreviousOutPoint
			txOut := wire.TxOut{Value: int64(v.Amount), PkScript: pkScript}
			err = s.PsbtUpdater.AddInWitnessUtxo(&txOut, v.Index)
			if err != nil {
				return err
			}
			err = s.PsbtUpdater.AddInSighashType(v.SighashType, v.Index)
			if err != nil {
				return err
			}
			prevOutputFetcher.AddPrevOut(outPoint, &txOut)
			sigHashes := txscript.NewTxSigHashes(s.PsbtUpdater.Upsbt.UnsignedTx, prevOutputFetcher)

			if v.RedeemScript != "" {
				redeemScript, err := hex.DecodeString(v.RedeemScript)
				if err != nil {
					return err
				}
				witnessArray, err := txscript.CalcTapscriptSignaturehash(sigHashes,
					v.SighashType, s.PsbtUpdater.Upsbt.UnsignedTx, v.Index, prevOutputFetcher, txscript.NewBaseTapLeaf(redeemScript))
				if err != nil {
					return err
				}
				signature, err := schnorr.Sign(privateKey, witnessArray)
				if err != nil {
					return err
				}
				taprootKeySpendSig = signature.Serialize()

				baseTapLeaf := txscript.NewBaseTapLeaf(redeemScript)
				targetLeafHash := baseTapLeaf.TapHash()
				newTaprootScriptSpendSig := make([]*psbt.TaprootScriptSpendSig, 0)
				xOnlyPubKey := schnorr.SerializePubKey(privateKey.PubKey())
				newTaprootScriptSpendSig = append(newTaprootScriptSpendSig, &psbt.TaprootScriptSpendSig{
					XOnlyPubKey: xOnlyPubKey,
					LeafHash:    targetLeafHash.CloneBytes(),
					Signature:   taprootKeySpendSig,
					SigHash:     v.SighashType,
				})
				s.PsbtUpdater.Upsbt.Inputs[v.Index].TaprootScriptSpendSig = newTaprootScriptSpendSig

				controlBlock, err := hex.DecodeString(v.ControlBlockWitness)
				if err != nil {
					return err
				}
				newTaprootLeafScript := make([]*psbt.TaprootTapLeafScript, 0)
				newTaprootLeafScript = append(newTaprootLeafScript, &psbt.TaprootTapLeafScript{
					ControlBlock: controlBlock,
					Script:       baseTapLeaf.Script,
					LeafVersion:  baseTapLeaf.LeafVersion,
				})
				s.PsbtUpdater.Upsbt.Inputs[v.Index].TaprootLeafScript = newTaprootLeafScript
			} else {
				witness, err := txscript.TaprootWitnessSignature(s.PsbtUpdater.Upsbt.UnsignedTx, sigHashes,
					v.Index, s.PsbtUpdater.Upsbt.Inputs[v.Index].WitnessUtxo.Value,
					s.PsbtUpdater.Upsbt.Inputs[v.Index].WitnessUtxo.PkScript,
					v.SighashType, privateKey)
				if err != nil {
					return err
				}
				taprootKeySpendSig = witness[0]
				s.PsbtUpdater.Upsbt.Inputs[v.Index].TaprootKeySpendSig = taprootKeySpendSig
			}

			fmt.Printf("TaprootWitnessSignature[%d]: %s\n", v.Index, hex.EncodeToString(taprootKeySpendSig))
			break
		}
		//fmt.Printf("index:%d\n, pri:%s\n, pub:%s\n, sigScript: %s\n", v.Index, v.PriHex, publicKey, hex.EncodeToString(sigScript))
		_, err = psbt.MaybeFinalize(s.PsbtUpdater.Upsbt, v.Index)
		if err != nil {
			fmt.Printf("Index-[%d] %s\n", v.Index, s.PsbtUpdater.Upsbt.UnsignedTx.TxIn[v.Index].PreviousOutPoint.String())
			return errors.New(fmt.Sprintf("Index-[%d] %s", v.Index, err))
		}
	}
	return nil
}

// add InputWitness with signed
func (s *PsbtBuilder) UpdateAndSignInput(signIns []*InputSign) error {
	var witnessScript []byte
	var pubByte []byte
	var redeemScript []byte
	multiPrevOutputFetcher := txscript.NewMultiPrevOutFetcher(nil)
	for i, txIn := range s.PsbtUpdater.Upsbt.UnsignedTx.TxIn {
		outPoint := txIn.PreviousOutPoint
		txOut := s.PsbtUpdater.Upsbt.Inputs[i].WitnessUtxo
		if txOut == nil {
			if s.PsbtUpdater.Upsbt.Inputs[i].NonWitnessUtxo != nil {
				txOut = s.PsbtUpdater.Upsbt.Inputs[i].NonWitnessUtxo.TxOut[outPoint.Index]
			}
		}
		multiPrevOutputFetcher.AddPrevOut(outPoint, txOut)
	}

	for _, v := range signIns {
		if v.PkScript != "" {
			pkScript, err := hex.DecodeString(v.PkScript)
			if err != nil {
				return err
			}
			outPoint := s.PsbtUpdater.Upsbt.UnsignedTx.TxIn[v.Index].PreviousOutPoint
			txOut := wire.TxOut{Value: int64(v.Amount), PkScript: pkScript}
			err = s.PsbtUpdater.AddInWitnessUtxo(&txOut, v.Index)
			if err != nil {
				return err
			}
			err = s.PsbtUpdater.AddInSighashType(v.SighashType, v.Index)
			if err != nil {
				return err
			}
			multiPrevOutputFetcher.AddPrevOut(outPoint, &txOut)
		}
	}

	for _, v := range signIns {
		privateKeyBytes, err := hex.DecodeString(v.PriHex)
		if err != nil {
			return err
		}
		privateKey, _ := btcec.PrivKeyFromBytes(privateKeyBytes)
		sigScript := []byte{}
		pubByte = privateKey.PubKey().SerializeCompressed()
		if v.RedeemScript != "" {
			redeemScript, err = hex.DecodeString(v.RedeemScript)
			if err != nil {
				return err
			}
		}
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
		case Taproot:
			pkScript, err := hex.DecodeString(v.PkScript)
			if err != nil {
				return err
			}

			outPoint := s.PsbtUpdater.Upsbt.UnsignedTx.TxIn[v.Index].PreviousOutPoint
			txOut := wire.TxOut{Value: int64(v.Amount), PkScript: pkScript}
			err = s.PsbtUpdater.AddInWitnessUtxo(&txOut, v.Index)
			if err != nil {
				return err
			}
			err = s.PsbtUpdater.AddInSighashType(v.SighashType, v.Index)
			if err != nil {
				return err
			}
			multiPrevOutputFetcher.AddPrevOut(outPoint, &txOut)
			sigHashes := txscript.NewTxSigHashes(s.PsbtUpdater.Upsbt.UnsignedTx, multiPrevOutputFetcher)
			witness, err := txscript.TaprootWitnessSignature(s.PsbtUpdater.Upsbt.UnsignedTx, sigHashes,
				v.Index, s.PsbtUpdater.Upsbt.Inputs[v.Index].WitnessUtxo.Value,
				s.PsbtUpdater.Upsbt.Inputs[v.Index].WitnessUtxo.PkScript,
				v.SighashType, privateKey)
			if err != nil {
				return err
			}
			witnessScript = witness[0]
			break
		}

		if v.UtxoType == Taproot {
			s.PsbtUpdater.Upsbt.Inputs[v.Index].TaprootKeySpendSig = witnessScript
		} else {
			res, err := s.PsbtUpdater.Sign(v.Index, sigScript, pubByte, redeemScript, nil)
			if err != nil || res != 0 {
				fmt.Printf("Index-[%d] %s  %s, SignOutcome:%d\n", v.Index, s.PsbtUpdater.Upsbt.UnsignedTx.TxIn[v.Index].PreviousOutPoint.String(), err, res)
				return errors.New(fmt.Sprintf("Sign:Index-[%d] %s, SignOutcome:%d", v.Index, err, res))
			}
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
		var redeemScript []byte
		//fmt.Printf("UpdateAndSignInput - signIn: %+v\n", v)
		privateKeyBytes, err := hex.DecodeString(v.PriHex)
		if err != nil {
			return err
		}
		privateKey, _ := btcec.PrivKeyFromBytes(privateKeyBytes)
		sigScript := []byte{}
		if v.RedeemScript != "" {
			redeemScript, err = hex.DecodeString(v.RedeemScript)
			if err != nil {
				return err
			}
		}
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
			subScript := s.PsbtUpdater.Upsbt.Inputs[v.Index].WitnessUtxo.PkScript
			if redeemScript != nil {
				subScript = redeemScript
			}
			prevOutputFetcher := NewPrevOutputFetcher(s.PsbtUpdater.Upsbt.Inputs[v.Index].WitnessUtxo.PkScript, s.PsbtUpdater.Upsbt.Inputs[v.Index].WitnessUtxo.Value)
			sigHashes := txscript.NewTxSigHashes(s.PsbtUpdater.Upsbt.UnsignedTx, prevOutputFetcher)
			sigScript, err = txscript.RawTxInWitnessSignature(s.PsbtUpdater.Upsbt.UnsignedTx, sigHashes,
				v.Index, s.PsbtUpdater.Upsbt.Inputs[v.Index].WitnessUtxo.Value,
				subScript,
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

func (s *PsbtBuilder) AddSigInForNonWitnessUtxo(nonWitnessUtxo *wire.MsgTx, partialSigs []*psbt.PartialSig, sighashType txscript.SigHashType, finalScriptSig []byte, index int) error {
	s.PsbtUpdater.Upsbt.Inputs[index].PartialSigs = partialSigs
	s.PsbtUpdater.Upsbt.Inputs[index].SighashType = sighashType
	s.PsbtUpdater.Upsbt.Inputs[index].NonWitnessUtxo = nonWitnessUtxo
	if err := s.PsbtUpdater.Upsbt.SanityCheck(); err != nil {
		return err
	}
	res, err := s.PsbtUpdater.Sign(index, partialSigs[0].Signature, partialSigs[0].PubKey, nil, nil)
	if err != nil || res != 0 {
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

	multiPrevOutputFetcher := txscript.NewMultiPrevOutFetcher(nil)
	if signIn.UtxoType == Taproot {
		for i, txIn := range s.PsbtUpdater.Upsbt.UnsignedTx.TxIn {
			outPoint := txIn.PreviousOutPoint
			txOut := s.PsbtUpdater.Upsbt.Inputs[i].WitnessUtxo
			multiPrevOutputFetcher.AddPrevOut(outPoint, txOut)
		}
	}

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
	case Taproot:
		pkScript, err := hex.DecodeString(signIn.PkScript)
		if err != nil {
			return err
		}

		outPoint := s.PsbtUpdater.Upsbt.UnsignedTx.TxIn[signIn.Index].PreviousOutPoint
		txOut := wire.TxOut{Value: int64(signIn.Amount), PkScript: pkScript}
		err = s.PsbtUpdater.AddInWitnessUtxo(&txOut, signIn.Index)
		if err != nil {
			return err
		}
		err = s.PsbtUpdater.AddInSighashType(signIn.SighashType, signIn.Index)
		if err != nil {
			return err
		}
		multiPrevOutputFetcher.AddPrevOut(outPoint, &txOut)
		fmt.Printf("multiPrevOutputFetcher: %s, %s\n", outPoint.String(), hex.EncodeToString(txOut.PkScript))

		for i, txIn := range s.PsbtUpdater.Upsbt.UnsignedTx.TxIn {
			outPoint := txIn.PreviousOutPoint
			txOut := multiPrevOutputFetcher.FetchPrevOutput(outPoint)
			fmt.Printf("multiPrevOutputFetcher[%d]: %s\n", i, outPoint.String())
			fmt.Printf("multiPrevOutputFetcher[%d]: %s\n", i, hex.EncodeToString(txOut.PkScript))
		}
		sigHashes := txscript.NewTxSigHashes(s.PsbtUpdater.Upsbt.UnsignedTx, multiPrevOutputFetcher)
		witness, err := txscript.TaprootWitnessSignature(s.PsbtUpdater.Upsbt.UnsignedTx, sigHashes,
			signIn.Index, s.PsbtUpdater.Upsbt.Inputs[signIn.Index].WitnessUtxo.Value,
			s.PsbtUpdater.Upsbt.Inputs[signIn.Index].WitnessUtxo.PkScript,
			signIn.SighashType, privateKey)
		if err != nil {
			return err
		}
		sigScript = witness[0]
		break
	}
	_ = witnessScript
	publicKey := hex.EncodeToString(privateKey.PubKey().SerializeCompressed())
	pubByte, err := hex.DecodeString(publicKey)
	if err != nil {
		return err
	}
	if signIn.UtxoType == Taproot {
		s.PsbtUpdater.Upsbt.Inputs[signIn.Index].TaprootKeySpendSig = sigScript
	} else {
		res, err := s.PsbtUpdater.Sign(signIn.Index, sigScript, pubByte, nil, nil)
		if err != nil || res != 0 {
			return err
		}
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

func (s *PsbtBuilder) AddInputByIndex(in Input, signIn *InputSign, index int64) error {

	multiPrevOutputFetcher := txscript.NewMultiPrevOutFetcher(nil)
	if signIn.UtxoType == Taproot {
		for i, txIn := range s.PsbtUpdater.Upsbt.UnsignedTx.TxIn {
			outPoint := txIn.PreviousOutPoint
			txOut := s.PsbtUpdater.Upsbt.Inputs[i].WitnessUtxo
			multiPrevOutputFetcher.AddPrevOut(outPoint, txOut)
		}
	}

	privateKeyBytes, err := hex.DecodeString(signIn.PriHex)
	if err != nil {
		return err
	}
	privateKey, _ := btcec.PrivKeyFromBytes(privateKeyBytes)
	publicKey := hex.EncodeToString(privateKey.PubKey().SerializeCompressed())
	pubByte, err := hex.DecodeString(publicKey)
	if err != nil {
		return err
	}
	sigScript := []byte{}
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
	case Taproot:
		pkScript, err := hex.DecodeString(signIn.PkScript)
		if err != nil {
			return err
		}

		outPoint := s.PsbtUpdater.Upsbt.UnsignedTx.TxIn[signIn.Index].PreviousOutPoint
		txOut := wire.TxOut{Value: int64(signIn.Amount), PkScript: pkScript}
		err = s.PsbtUpdater.AddInWitnessUtxo(&txOut, signIn.Index)
		if err != nil {
			return err
		}
		err = s.PsbtUpdater.AddInSighashType(signIn.SighashType, signIn.Index)
		if err != nil {
			return err
		}
		multiPrevOutputFetcher.AddPrevOut(outPoint, &txOut)

		//prevOutputFetcher := txscript.NewCannedPrevOutputFetcher(s.PsbtUpdater.Upsbt.Inputs[signIn.Index].WitnessUtxo.PkScript, s.PsbtUpdater.Upsbt.Inputs[signIn.Index].WitnessUtxo.Value)

		sigHashes := txscript.NewTxSigHashes(s.PsbtUpdater.Upsbt.UnsignedTx, multiPrevOutputFetcher)
		//sigHashes := txscript.NewTxSigHashes(s.PsbtUpdater.Upsbt.UnsignedTx, prevOutputFetcher)
		witness, err := txscript.TaprootWitnessSignature(s.PsbtUpdater.Upsbt.UnsignedTx, sigHashes,
			signIn.Index, s.PsbtUpdater.Upsbt.Inputs[signIn.Index].WitnessUtxo.Value,
			s.PsbtUpdater.Upsbt.Inputs[signIn.Index].WitnessUtxo.PkScript,
			signIn.SighashType, privateKey)
		if err != nil {
			return err
		}
		sigScript = witness[0]
		break
	}

	if signIn.UtxoType == Taproot {
		s.PsbtUpdater.Upsbt.Inputs[signIn.Index].TaprootKeySpendSig = sigScript
	} else {
		res, err := s.PsbtUpdater.Sign(signIn.Index, sigScript, pubByte, nil, nil)
		if err != nil || res != 0 {
			return err
		}
	}
	_, err = psbt.MaybeFinalize(s.PsbtUpdater.Upsbt, signIn.Index)
	if err != nil {
		return err
	}
	return nil
}
func (s *PsbtBuilder) AddInputOnly(in Input) error {
	txHash, err := chainhash.NewHashFromStr(in.OutTxId)
	if err != nil {
		return err
	}
	s.PsbtUpdater.Upsbt.UnsignedTx.AddTxIn(&wire.TxIn{
		PreviousOutPoint: *wire.NewOutPoint(txHash, in.OutIndex),
		Sequence:         wire.MaxTxInSequenceNum,
	})
	s.PsbtUpdater.Upsbt.Inputs = append(s.PsbtUpdater.Upsbt.Inputs, psbt.PInput{})
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

func (s *PsbtBuilder) CalTxSize() (int64, error) {
	var (
		tx          *wire.MsgTx = s.PsbtUpdater.Upsbt.UnsignedTx
		txTotalSize int         = tx.SerializeSize()
		txBaseSize  int         = tx.SerializeSizeStripped()
		weight      int64       = 0
		vSize       int64       = 0

		pIns []psbt.PInput = s.PsbtUpdater.Upsbt.Inputs

		emptySegwitWitenss   = wire.TxWitness{make([]byte, 71), make([]byte, 33)}
		emptyNestSignature   = make([]byte, 23)
		emptylegacySignature = make([]byte, 107)
		emptyTaprootWitness  = wire.TxWitness{make([]byte, 64)}
	)

	for _, v := range pIns {
		if v.WitnessUtxo != nil {
			if v.RedeemScript != nil {
				txBaseSize += 40 + wire.VarIntSerializeSize(uint64(len(emptyNestSignature))) + len(emptyNestSignature)
			}
			if v.TaprootKeySpendSig != nil {
				txTotalSize += emptyTaprootWitness.SerializeSize()
			} else {
				txTotalSize += emptySegwitWitenss.SerializeSize()
			}
		} else if v.NonWitnessUtxo != nil {
			txBaseSize += 40 + wire.VarIntSerializeSize(uint64(len(emptylegacySignature))) + len(emptylegacySignature)
		}
	}

	weight = int64(txBaseSize*3 + txTotalSize)
	vSize = (weight + (blockchain.WitnessScaleFactor - 1)) / blockchain.WitnessScaleFactor
	return vSize, nil
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
