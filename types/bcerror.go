package types

//BcError structure of bcerror
type BcError struct {
	ErrorCode uint32 // Error code
	ErrorDesc string // Error description
}

// Error() gets error description with error code
func (bcerror *BcError) Error() string {
	if bcerror.ErrorDesc != "" {
		return bcerror.ErrorDesc
	}

	for _, error := range bcErrors {
		if error.ErrorCode == bcerror.ErrorCode {
			return error.ErrorDesc
		}
	}
	return ""
}

//CodeOK means success
// CodeBVMQueryOK success of call BVM view function
const (
	CodeOK         = 200 + iota
	CodeBVMQueryOK = 201
)

// internal error code
const (
	ErrInternalFailed = 500 + iota
	ErrDealFailed
	ErrPath
	ErrMarshal
	ErrCallRPC
	ErrAccountLocked
)

//ErrCheckTx beginning error code of checkTx
const (
	ErrCheckTx = 600 + iota
)

//ErrDeliverTx beginning error code of deliverTx
const (
	ErrDeliverTx = 700 + iota
)

const (
	ErrNoAuthorization = 1000 + iota
)

// ErrCodeBVMInvoke beginning error code of BVM execution
const (
	ErrCodeBVMInvoke = 3000 + iota
	ErrRlpDecode
	ErrCallState
	ErrFeeNotEnough
	ErrInvalidParam
)

const (
	ErrLogicError = 5000 + iota
)

var bcErrors = []BcError{
	{CodeOK, ""},

	{ErrInternalFailed, "Internal failed"},
	{ErrDealFailed, "Deal failed"},
	{ErrPath, "Invalid url path"},

	{ErrCheckTx, "CheckTx failed"},

	//ErrCodeNoAuthorization
	{ErrNoAuthorization, "No authorization"},

	{ErrDeliverTx, "DeliverTx failed"},
	{ErrMarshal, "Json marshal error"},
	{ErrCallRPC, "Call rpc error"},
	{ErrDealFailed, "The deal failed"},
	{ErrAccountLocked, "Account is locked"},

	{ErrCheckTx, "CheckTx failed"},

	{ErrDeliverTx, "DeliverTx failed"},

	// Err describe of BVM
	{ErrCodeBVMInvoke, "BVM invokeTx failed"},
	{ErrRlpDecode, "BVM rlp decode failed"},
	{ErrCallState, "BVM of error calling state"},
	{ErrFeeNotEnough, "BVM Insufficient balance to pay fee"},
	{ErrInvalidParam, "BVM Param is wrong"},
}
