package psbt_sdk

import (
	"bytes"
	"encoding/hex"
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
	NetParams *chaincfg.Params
	PsbtPacket *psbt.Packet
	PsbtUpdater *psbt.Updater
	PsbtRawTx []byte
}

func (s *PsbtBuilder) CreatePsbtTransaction(ins []Input, outs []Output) error {
	var(
		txOuts []*wire.TxOut = make([]*wire.TxOut, 0)
		txIns []*wire.OutPoint = make([]*wire.OutPoint, 0)
		nSequences []uint32 = make([]uint32, 0)
	)
	for _, in := range ins{
		txHash, err := chainhash.NewHashFromStr(in.OutTxId)
		if err != nil {
			return err
		}
		prevOut := wire.NewOutPoint(txHash, in.OutIndex)
		txIns = append(txIns, prevOut)
		nSequences = append(nSequences, wire.MaxTxInSequenceNum)
	}

	for _, out := range outs {
		address, err := btcutil.DecodeAddress(out.Address, s.NetParams)
		if err != nil {
			return err
		}

		pkScript, err := txscript.PayToAddrScript(address)
		if err != nil {
			return err
		}

		txOut := wire.NewTxOut(int64(out.Amount), pkScript)
		txOuts = append(txOuts, txOut)
	}

	cPsbt, err := psbt.New(txIns, txOuts, int32(2), uint32(0), nSequences)
	if err != nil {
		return err
	}
	s.PsbtPacket = cPsbt
	s.PsbtUpdater, err = psbt.NewUpdater(s.PsbtPacket)
	if err != nil {
		return err
	}
	return nil
}

func (s *PsbtBuilder) UpdatePsbtTransaction(inUtxos []InputUtxo) error {
	for _, v := range inUtxos {
		switch v.UtxoType {
		case NonWitness:
			tx := wire.NewMsgTx(2)
			nonWitnessUtxoHex, err := hex.DecodeString(v.NonWitnessUtxo)
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
		case Witness:
			witnessUtxoScriptHex, err := hex.DecodeString(
				v.WitnessUtxoPkScript)
			if err != nil {
				return err
			}
			txout := wire.TxOut{Value: int64(v.WitnessUtxoAmount), PkScript: witnessUtxoScriptHex}
			err = s.PsbtUpdater.AddInWitnessUtxo(&txout, v.Index)
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

func (s *PsbtBuilder) SignPsbtTransaction(inSigners []InputSigner) error {
	for _, v := range inSigners {
		//sigByte, err := hex.DecodeString(v.Sig)
		privateKeyBytes, err := hex.DecodeString(v.Pri)
		if err != nil {
			return err
		}
		privateKey, _ := btcec.PrivKeyFromBytes(privateKeyBytes)

		sigScript := []byte{}
		switch v.UtxoType {
		case NonWitness:
			sigScript, err = txscript.RawTxInSignature(s.PsbtUpdater.Upsbt.UnsignedTx, v.Index, s.PsbtUpdater.Upsbt.Inputs[v.Index].NonWitnessUtxo.TxOut[s.PsbtUpdater.Upsbt.UnsignedTx.TxIn[v.Index].PreviousOutPoint.Index].PkScript, v.SighashType, privateKey)
			if err != nil {
				return err
			}
			break
		case Witness:
			prevOutputFetcher := NewPrevOutputFetcher(s.PsbtUpdater.Upsbt.Inputs[v.Index].WitnessUtxo.PkScript, s.PsbtUpdater.Upsbt.Inputs[v.Index].WitnessUtxo.Value)
			sigHashes := txscript.NewTxSigHashes(s.PsbtUpdater.Upsbt.UnsignedTx, prevOutputFetcher)
			sigScript, err = txscript.RawTxInWitnessSignature(s.PsbtUpdater.Upsbt.UnsignedTx, sigHashes, v.Index, s.PsbtUpdater.Upsbt.Inputs[v.Index].WitnessUtxo.Value, s.PsbtUpdater.Upsbt.Inputs[v.Index].WitnessUtxo.PkScript, v.SighashType, privateKey)
			if err != nil {
				return err
			}
			break
		}

		fmt.Printf("sigScript: %s\n", hex.EncodeToString(sigScript))
		pubByte, err := hex.DecodeString(v.Pub)
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

func (s *PsbtBuilder) NewUpdaterFromPsbtTransaction(rawTx string) error {

	return nil
}

func (s *PsbtBuilder) IsComplete() bool {
	return s.PsbtPacket.IsComplete()
}


func (s *PsbtBuilder) ExtractPsbtTransaction() (string, error) {

	if !s.IsComplete() {
		err := psbt.MaybeFinalizeAll(s.PsbtPacket)
		if err != nil {
			return "", err
		}
	}


	tx, err := psbt.Extract(s.PsbtPacket)
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


type PrevOutputFetcher struct{
	pkScript []byte
	value int64
}


func NewPrevOutputFetcher(pkScript []byte, value int64) *PrevOutputFetcher {
	return &PrevOutputFetcher{
		pkScript,
		value,
	}
}


func (d *PrevOutputFetcher) FetchPrevOutput(wire.OutPoint) *wire.TxOut{
	return &wire.TxOut{
		Value:    d.value,
		PkScript: d.pkScript,
	}
}