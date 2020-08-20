package types

import (
	"github.com/bcbchain/bclib/tendermint/go-crypto"
	tx1 "github.com/bcbchain/bclib/tx/v1"
	type2 "github.com/bcbchain/bclib/types"
)

type TxV1Result struct {
	FromAddr    crypto.Address
	Pubkey      crypto.PubKeyEd25519
	Transaction tx1.Transaction
	BFailed     bool
}

type TxV2Result struct {
	Pubkey      crypto.PubKeyEd25519
	Transaction type2.Transaction
}

type TxV3Result struct {
	Pubkey      crypto.PubKeyEd25519
	Transaction type2.Transaction
}

type BCError struct {
	ErrorCode uint32 // Error code
	ErrorDesc string // Error description
}
