package rpcclient

import (
	amino "github.com/bcbchain/bclib/tendermint/go-amino"
	crypto "github.com/bcbchain/bclib/tendermint/go-crypto"
)

var CDC = amino.NewCodec()

func init() {
	crypto.RegisterAmino(CDC)
}
