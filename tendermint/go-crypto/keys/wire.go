package keys

import (
	amino "github.com/bcbchain/bclib/tendermint/go-amino"
	crypto "github.com/bcbchain/bclib/tendermint/go-crypto"
)

var cdc = amino.NewCodec()

func init() {
	crypto.RegisterAmino(cdc)
}
