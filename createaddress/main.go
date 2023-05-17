package main

import (
	"encoding/hex"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"log"
)

func main() {
	netParams := &chaincfg.SigNetParams
	//netParams := &chaincfg.TestNet3Params
	privateKey, err := btcec.NewPrivateKey()
	if err != nil {
		log.Fatal(err)
	}
	privateKeyHex := hex.EncodeToString(privateKey.Serialize())
	log.Printf("new priviate key %s \n", privateKeyHex)

	publicKey := hex.EncodeToString(privateKey.PubKey().SerializeCompressed())
	log.Printf("new public key %s \n", publicKey)

	legacyAddress, err := btcutil.NewAddressPubKey(privateKey.PubKey().SerializeCompressed(), netParams)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("new legacy address %s \n", legacyAddress.EncodeAddress())


	nativeSegwitAddress, err := btcutil.NewAddressWitnessPubKeyHash(btcutil.Hash160(privateKey.PubKey().SerializeCompressed()), netParams)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("new native segwit address %s \n", nativeSegwitAddress.EncodeAddress())


	taprootAddress, err := btcutil.NewAddressTaproot(schnorr.SerializePubKey(txscript.ComputeTaprootKeyNoScript(privateKey.PubKey())), netParams)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("new taproot address %s \n", taprootAddress.EncodeAddress())

	restorePrivateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		log.Fatal(err)
	}
	restorePrivateKey, _ := btcec.PrivKeyFromBytes(restorePrivateKeyBytes)

	restoreTaprootAddress, err := btcutil.NewAddressTaproot(schnorr.SerializePubKey(txscript.ComputeTaprootKeyNoScript(restorePrivateKey.PubKey())), netParams)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("restore taproot address %s \n", restoreTaprootAddress.EncodeAddress())

	if taprootAddress.EncodeAddress() != restoreTaprootAddress.EncodeAddress() {
		log.Fatal("restore privateKey error")
	}
	/**
	test btc faucet
	https://signetfaucet.com/
	https://alt.signetfaucet.com/
	*/
}
