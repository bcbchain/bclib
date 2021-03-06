package tx3

import (
	"encoding/base64"
	"encoding/hex"
	"strings"

	"github.com/bcbchain/bclib/rlp"
	"github.com/bcbchain/bclib/sig"
	"github.com/bcbchain/bclib/tendermint/go-crypto"
	"github.com/bcbchain/bclib/tendermint/tmlibs/common"
	"github.com/bcbchain/bclib/types"
	"github.com/bcbchain/bclib/wal"
)

var (
	gChainID string
)

// Init - chainID
func Init(_chainID string) {
	gChainID = _chainID
}

// WrapInvokeParams - wrap contract parameters
func WrapInvokeParams(params ...interface{}) []common.HexBytes {
	paramsRlp := make([]common.HexBytes, len(params))
	for i, param := range params {
		var paramRlp []byte
		var err error

		paramRlp, err = rlp.EncodeToBytes(param)
		if err != nil {
			panic(err)
		}
		paramsRlp[i] = paramRlp
	}

	return paramsRlp
}

// WrapPayload - wrap contracts to payload byte
func WrapPayload(nonce uint64, gasLimit int64, note string, messages ...types.Message) []byte {

	type transaction struct {
		Nonce    uint64
		GasLimit int64
		Note     string
		Messages []types.Message
	}
	tx := transaction{
		Nonce:    nonce,
		GasLimit: gasLimit,
		Note:     note,
		Messages: messages,
	}
	txRlp, err := rlp.EncodeToBytes(tx)
	if err != nil {
		panic(err)
	}
	return txRlp
}

// WrapTx - sign the payload to string
// privateKey的格式:
// name:password
// enprivatekey:password
// 0x十六进制表示的私钥数据
//nolint unhandled
func WrapTx(payload []byte, privateKey string) string {
	return WrapTxEx(gChainID, payload, privateKey)
}

func WrapTxEx(chainID string, payload []byte, privateKey string) string {
	var sigInfo sig.Ed25519Sig
	var isHexPrivKey = strings.HasPrefix(privateKey, "0x")
	var segPrivKey = strings.Split(privateKey, ":")

	if isHexPrivKey && len(segPrivKey) == 1 {
		var privKey crypto.PrivKey
		var pubKey crypto.PubKey

		hexData := privateKey[2:]
		privKeyBytes, err := hex.DecodeString(hexData)
		if err != nil {
			panic(err.Error())
		}
		privKey = crypto.PrivKeyEd25519FromBytes(privKeyBytes)
		pubKey = privKey.PubKey()

		sigInfo = sig.Ed25519Sig{
			SigType:  "ed25519",
			PubKey:   pubKey.(crypto.PubKeyEd25519),
			SigValue: privKey.Sign(payload).(crypto.SignatureEd25519),
		}
	} else if len(segPrivKey) == 2 {
		acct, err := wal.LoadAccount(".", segPrivKey[0], segPrivKey[1])
		if err != nil {
			panic(err.Error())
		}
		si, err := acct.Sign(payload)
		if err != nil {
			panic(err.Error())
		}
		sigInfo = *si
	} else {
		panic("Invalid private key format")
	}

	size, r, err := rlp.EncodeToReader(sigInfo)
	if err != nil {
		panic(err.Error())
	}
	sig := make([]byte, size)
	r.Read(sig)

	payloadString := base64.StdEncoding.EncodeToString(payload)
	sigString := base64.StdEncoding.EncodeToString(sig)

	MAC := chainID + "<tx>"
	Version := "v3"
	SignerNumber := "<1>"

	return MAC + "." + Version + "." + payloadString + "." + SignerNumber + "." + sigString
}
