package tx1

import (
	"bytes"
	"errors"
	"strings"

	"github.com/bcbchain/bclib/rlp"
	"github.com/bcbchain/bclib/tendermint/go-crypto"
	"github.com/btcsuite/btcutil/base58"
)

// 定义解析一笔交易（包含签名验证）的接口函数，将结果填入Transaction数据结构，其中Data字段为RLP编码的合约调用参数
func (tx *Transaction) TxParse(chainID, txString string) (crypto.Address, crypto.PubKeyEd25519, error) {
	MAC := chainID + "<tx>"
	Version := "v1"
	SignerNumber := "<1>"
	strs := strings.Split(txString, ".")

	if len(strs) != 5 {
		return "", crypto.PubKeyEd25519{}, errors.New("tx data error")
	}

	if strs[0] != MAC || strs[1] != Version || strs[3] != SignerNumber {
		return "", crypto.PubKeyEd25519{}, errors.New("tx data error")
	}

	txData := base58.Decode(strs[2])
	sigBytes := base58.Decode(strs[4])

	reader := bytes.NewReader(sigBytes)
	var siginfo Ed25519Sig
	err := rlp.Decode(reader, &siginfo)
	if err != nil {
		return "", crypto.PubKeyEd25519{}, err
	}

	if !siginfo.PubKey.VerifyBytes(txData, siginfo.SigValue) {
		return "", siginfo.PubKey, errors.New("verify sig fail")
	}

	//RLP解码Transaction结构
	reader = bytes.NewReader(txData)
	err = rlp.Decode(reader, tx)
	if err != nil {
		return "", siginfo.PubKey, err
	}

	//crypto.SetChainId(chainID)
	return siginfo.PubKey.Address(chainID), siginfo.PubKey, nil
}
