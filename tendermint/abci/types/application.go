package types // nolint: goimports

import (
	"golang.org/x/net/context"
)

// Application is an interface that enables any finite, deterministic state machine
// to be driven by a blockchain-based replication engine via the ABCI.
// All methods take a RequestXxx argument and return a ResponseXxx argument,
// except CheckTx/DeliverTx, which take `tx []byte`, and `Commit`, which takes nothing.
type Application interface {
	// Info/Query Connection
	Info(RequestInfo) ResponseInfo                // Return application info
	SetOption(RequestSetOption) ResponseSetOption // Set application option
	Query(RequestQuery) ResponseQuery             // Query for state
	QueryEx(RequestQueryEx) ResponseQueryEx       // QueryEx for state

	// Mempool Connection
	CheckTx(tx []byte) ResponseCheckTx // Validate a tx for the mempool
	CheckTxs(txs [][]byte) ResponseCheckTxs
	CheckTxConcurrency(tx []byte, responses chan<- *Response)
	// Consensus Connection
	InitChain(RequestInitChain) ResponseInitChain    // Initialize blockchain with validators and other info from TendermintCore
	BeginBlock(RequestBeginBlock) ResponseBeginBlock // Signals the beginning of a block
	DeliverTx(tx []byte) ResponseDeliverTx           // Deliver a tx for full processing
	DeliverTxs(txs [][]byte) ResponseDeliverTxs
	EndBlock(RequestEndBlock) ResponseEndBlock // Signals the end of a block, returns changes to the validator set
	Commit() ResponseCommit                    // Commit the state and return the application Merkle root hash

	// Clear all bcchain data when side chain genesis
	CleanData() ResponseCleanData
	GetGenesis() ResponseGetGenesis
	Rollback() ResponseRollback
}

//-------------------------------------------------------
// BaseApplication is a base form of Application

var _ Application = (*BaseApplication)(nil)

type BaseApplication struct {
}

func NewBaseApplication() *BaseApplication {
	return &BaseApplication{}
}

func (BaseApplication) Info(req RequestInfo) ResponseInfo {
	return ResponseInfo{}
}

func (BaseApplication) SetOption(req RequestSetOption) ResponseSetOption {
	return ResponseSetOption{Code: CodeTypeOK}
}

func (BaseApplication) DeliverTx(tx []byte) ResponseDeliverTx {
	return ResponseDeliverTx{Code: CodeTypeOK}
}
func (BaseApplication) DeliverTxs(txs [][]byte) ResponseDeliverTxs {
	DeliverTxs := []ResponseDeliverTx{}
	for i, _ := range txs {
		DeliverTxs[i] = ResponseDeliverTx{Code: CodeTypeOK}
	}
	responseDeliverTxs := ResponseDeliverTxs{DeliverTxs}
	return responseDeliverTxs
}

func (BaseApplication) CheckTx(tx []byte) ResponseCheckTx {
	return ResponseCheckTx{Code: CodeTypeOK}
}
func (BaseApplication) CheckTxs(txs [][]byte) ResponseCheckTxs {
	CheckTxs := []ResponseCheckTx{}
	for i, _ := range txs {
		CheckTxs[i] = ResponseCheckTx{Code: CodeTypeOK}
	}
	responseCheckTxs := ResponseCheckTxs{
		CheckTxs,
	}
	return responseCheckTxs
}

func (BaseApplication) CheckTxConcurrency(tx []byte, responses chan<- *Response) {
	//panic("implement me")
}

func (BaseApplication) Commit() ResponseCommit {
	return ResponseCommit{}
}

func (BaseApplication) Query(req RequestQuery) ResponseQuery {
	return ResponseQuery{Code: CodeTypeOK}
}

func (BaseApplication) QueryEx(req RequestQueryEx) ResponseQueryEx {
	return ResponseQueryEx{Code: CodeTypeOK}
}

func (BaseApplication) InitChain(req RequestInitChain) ResponseInitChain {
	return ResponseInitChain{Code: CodeTypeOK}
}

func (BaseApplication) BeginBlock(req RequestBeginBlock) ResponseBeginBlock {
	return ResponseBeginBlock{Code: CodeTypeOK}
}

func (BaseApplication) EndBlock(req RequestEndBlock) ResponseEndBlock {
	return ResponseEndBlock{}
}

func (BaseApplication) CleanData() ResponseCleanData {
	return ResponseCleanData{}
}

func (BaseApplication) GetGenesis() ResponseGetGenesis {
	return ResponseGetGenesis{}
}

func (BaseApplication) Rollback() ResponseRollback {
	return ResponseRollback{}
}

//-------------------------------------------------------

// GRPCApplication is a GRPC wrapper for Application
type GRPCApplication struct {
	app Application
}

func NewGRPCApplication(app Application) *GRPCApplication {
	return &GRPCApplication{app}
}

func (app *GRPCApplication) CheckTxConcurrency(ctx context.Context, req *RequestCheckTx, resChan chan<- *Response) error {
	panic("implement me")
}

func (app *GRPCApplication) Echo(ctx context.Context, req *RequestEcho) (*ResponseEcho, error) {
	return &ResponseEcho{req.Message}, nil
}

func (app *GRPCApplication) Flush(ctx context.Context, req *RequestFlush) (*ResponseFlush, error) {
	return &ResponseFlush{}, nil
}

func (app *GRPCApplication) Info(ctx context.Context, req *RequestInfo) (*ResponseInfo, error) {
	res := app.app.Info(*req)
	return &res, nil
}

func (app *GRPCApplication) SetOption(ctx context.Context, req *RequestSetOption) (*ResponseSetOption, error) {
	res := app.app.SetOption(*req)
	return &res, nil
}

func (app *GRPCApplication) DeliverTxs(ctx context.Context, req *RequestDeliverTxs) (*ResponseDeliverTxs, error) {
	res := app.app.DeliverTxs(req.Txs)
	return &res, nil
}

func (app *GRPCApplication) DeliverTx(ctx context.Context, req *RequestDeliverTx) (*ResponseDeliverTx, error) {
	res := app.app.DeliverTx(req.Tx)
	return &res, nil
}

func (app *GRPCApplication) CheckTxs(ctx context.Context, req *RequestCheckTxs) (*ResponseCheckTxs, error) {
	res := app.app.CheckTxs(req.Txs)
	return &res, nil
}

func (app *GRPCApplication) CheckTx(ctx context.Context, req *RequestCheckTx) (*ResponseCheckTx, error) {
	res := app.app.CheckTx(req.Tx)
	return &res, nil
}

func (app *GRPCApplication) Query(ctx context.Context, req *RequestQuery) (*ResponseQuery, error) {
	res := app.app.Query(*req)
	return &res, nil
}

func (app *GRPCApplication) QueryEx(ctx context.Context, req *RequestQueryEx) (*ResponseQueryEx, error) {
	res := app.app.QueryEx(*req)
	return &res, nil
}

func (app *GRPCApplication) Commit(ctx context.Context, req *RequestCommit) (*ResponseCommit, error) {
	res := app.app.Commit()
	return &res, nil
}

func (app *GRPCApplication) InitChain(ctx context.Context, req *RequestInitChain) (*ResponseInitChain, error) {
	res := app.app.InitChain(*req)
	return &res, nil
}

func (app *GRPCApplication) BeginBlock(ctx context.Context, req *RequestBeginBlock) (*ResponseBeginBlock, error) {
	res := app.app.BeginBlock(*req)
	return &res, nil
}

func (app *GRPCApplication) EndBlock(ctx context.Context, req *RequestEndBlock) (*ResponseEndBlock, error) {
	res := app.app.EndBlock(*req)
	return &res, nil
}

func (app *GRPCApplication) CleanData(ctx context.Context, req *RequestCleanData) (*ResponseCleanData, error) {
	res := app.app.CleanData()
	return &res, nil
}

func (app *GRPCApplication) GetGenesis(ctx context.Context, req *RequestGetGenesis) (*ResponseGetGenesis, error) {
	res := app.app.GetGenesis()
	return &res, nil
}

func (app *GRPCApplication) Rollback(ctx context.Context, req *RequestRollback) (*ResponseRollback, error) {
	res := app.app.Rollback()
	return &res, nil
}
