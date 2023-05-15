package psbt_sdk

import (
	"bytes"
	"encoding/hex"
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

	for _, in := range ins{

		txHash, err := chainhash.NewHashFromStr(in.OutTxId)
		if err != nil {
			return err
		}
		prevOut := wire.NewOutPoint(txHash, in.OutIndex)
		txIns = append(txIns, prevOut)
		nSequences = append(nSequences, wire.MaxTxInSequenceNum)
	}

	// Use valid data to create:
	cPsbt, err := psbt.New(txIns, txOuts, int32(2), uint32(0), nSequences)
	//var b bytes.Buffer
	//err = cPsbt.Serialize(&b)
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
			nonWitnessUtxoHex, err := hex.DecodeString(
				v.NonWitnessUtxo)
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
			break
		case Witness:
			witnessUtxoHex, err := hex.DecodeString(
				v.WitnessUtxo)
			if err != nil {
				return err
			}
			txout := wire.TxOut{Value: int64(v.WitnessUtxoAmount),
				PkScript: witnessUtxoHex[9:]}

			err = s.PsbtUpdater.AddInWitnessUtxo(&txout, v.Index)
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
		sigByte, err := hex.DecodeString(v.Sig)
		pubByte, err := hex.DecodeString(v.Pub)
		res, err := s.PsbtUpdater.Sign(0, sigByte, pubByte, nil, nil)
		if err != nil || res != 0 {
			return err
		}
	}
	return nil
}

func (s *PsbtBuilder) PsbtTransactionToRawString() (string, error) {
	var b bytes.Buffer
	err := s.PsbtUpdater.Upsbt.Serialize(&b)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}