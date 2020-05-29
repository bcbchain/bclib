package hdwal

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/bcbchain/bclib/bcdb"
	"github.com/bcbchain/bclib/types"
	"github.com/bcbchain/bclib/tendermint/go-crypto"
	"testing"
)

func Test1_Mnemonic(t *testing.T) {
	db, _ := bcdb.OpenDB("./", "", "")
	SetDB(db)
	mnemonic, err := Mnemonic("12345rewq!")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("create mnemonic:", mnemonic)

	privKey1, err := NewPrivKey("m/44'/60'/0'/0/0", "12345rewq!")
	if err != nil {
		panic(err)
	}
	pubKey1 := privKey1.PubKey()
	PriK1 := privKey1.(crypto.PrivKeyEd25519)
	PubK1 := pubKey1.(crypto.PubKeyEd25519)

	fmt.Println("PrivateKey1: ", "0x"+hex.EncodeToString(PriK1[:]))
	fmt.Println("PubKey1:     ", "0x"+hex.EncodeToString(PubK1[:]))
	fmt.Println("address1:     ", PriK1.PubKey().Address("bcb"))
	mnemonic, err = Export("12345rewq!")
	if err != nil {
		panic(err)
	}

	fmt.Println("export mnemonic:", mnemonic)

	err = Import(mnemonic, "12345rewq!")
	if err != nil {
		panic(err)
	}

	privKey2, err := NewPrivKey("m/44'/60'/0'/0/0", "12345rewq!")
	if err != nil {
		panic(err)
	}

	pubKey2 := privKey2.PubKey()
	PriK2 := privKey2.(crypto.PrivKeyEd25519)
	PubK2 := pubKey2.(crypto.PubKeyEd25519)

	fmt.Println("PrivateKey2: ", "0x"+hex.EncodeToString(PriK2[:]))
	fmt.Println("PubKey2:     ", "0x"+hex.EncodeToString(PubK2[:]))
	fmt.Println("address:     ", PriK2.PubKey().Address("bcb"))
	if bytes.Compare(PriK2.Bytes(), PriK1.Bytes()) != 0 && bytes.Compare(PubK2.Bytes(), PubK1.Bytes()) != 0 {
		panic("PriK2 != PriK1 , PubK2 != PubK1")
	}
	if PriK2.PubKey().Address("bcb") != PriK2.PubKey().Address("bcb") {
		panic("1!=2")
	}
}

func Test2_Mnemonic(t *testing.T) {
	db, _ := bcdb.OpenDB("./", "", "")
	SetDB(db)

	mnemonic := "half then limit day ridge input craft street hammer smile reunion body"
	err := Import(mnemonic, "12345rewq!")
	if err != nil {
		panic(err)
	}
	var tests = []struct {
		path    string
		address types.Address
	}{
		{"m/44'/60'/0'/0/0", "bcbAGVHi1Xkwt7RuKbsTyA2u7MBUFtNqbB2K"},
		{"m/44'/60'/0'/0/1", "bcb9tpzii1SiEx8kmgMmJV518jUVoQk5XYgN"},
		{"m/44'/60'/0'/0/2", "bcbFkmzdFTAa2fQ3tXbqArguRknsswgSRBgQ"},
		{"m/44'/60'/0'/0/3", "bcb8i3UDNzJQocpE6xT28hvMutuXzB3jP4oh"},
		{"m/44'/60'/0'/0/4", "bcb8FpwnVTqrWeADvayfCHoz5Vvjs6c44RG8"},
		{"m/44'/60'/0'/0/5", "bcbP6zKYp9urPH9U7BLbHquaruR83QrHqt1h"},
		{"m/44'/60'/0'/0/6", "bcbJFtpnrZN9Le8pegSe7zo8tjRWeVBESEiC"},
		{"m/44'/60'/0'/0/7", "bcbKy4ouLAaWycBQY2RJyMefGPFeb6U789jr"},
		{"m/44'/60'/0'/0/8", "bcb3LTHcgKWYmHu5TRCDGFNXA88Tk4SoVkKo"},
		{"m/44'/60'/0'/0/9", "bcb8pqfzJPMa2jyHrFVwoC4Utav7qrRBDpXf"},
		{"m/44'/60'/0'/0/10", "bcb4fbYnL7oqXcytNHXSjN6cTzuoNyVyUb7f"},
	}
	for _, item := range tests {
		privKey, err := NewPrivKey(item.path, "12345rewq!")
		if err != nil {
			panic(err)
		}
		if item.address != privKey.PubKey().Address("bcb") {
			panic("地址不相等！")
		}
	}
	privKey, err := NewPrivKey("m/44'/60'/0", "12345rewq!")
	fmt.Println("------", err)
	fmt.Println("------", privKey)
}
