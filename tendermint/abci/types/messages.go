package types

import (
	"bufio"
	"encoding/binary"
	"io"

	"github.com/gogo/protobuf/proto"
)

const (
	maxMsgSize = 104857600 // 100MB
)

// WriteMessage writes a varint length-delimited protobuf message.
func WriteMessage(msg proto.Message, w io.Writer) error {
	bz, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	return encodeByteSlice(w, bz)
}

// ReadMessage reads a varint length-delimited protobuf message.
func ReadMessage(r io.Reader, msg proto.Message) error {
	return readProtoMsg(r, msg, maxMsgSize)
}

func readProtoMsg(r io.Reader, msg proto.Message, maxSize int) error {
	// binary.ReadVarint takes an io.ByteReader, eg. a bufio.Reader
	reader, ok := r.(*bufio.Reader)
	if !ok {
		reader = bufio.NewReader(r)
	}
	length64, err := binary.ReadVarint(reader)
	if err != nil {
		return err
	}
	length := int(length64)
	if length < 0 || length > maxSize {
		return io.ErrShortBuffer
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(reader, buf); err != nil {
		return err
	}
	//fmt.Println("收到的buf", string(buf))
	return proto.Unmarshal(buf, msg)
}

//-----------------------------------------------------------------------
// NOTE: we copied wire.EncodeByteSlice from go-wire rather than keep
// go-wire as a dep

func encodeByteSlice(w io.Writer, bz []byte) (err error) {
	err = encodeVarint(w, int64(len(bz)))
	if err != nil {
		return
	}
	_, err = w.Write(bz)
	return
}

func encodeVarint(w io.Writer, i int64) (err error) {
	var buf [10]byte
	n := binary.PutVarint(buf[:], i)
	_, err = w.Write(buf[0:n])
	return
}

//----------------------------------------

func ToRequestEcho(message string) *Request {
	return &Request{
		Value: &Request_Echo{&RequestEcho{message}},
	}
}

func ToRequestFlush() *Request {
	return &Request{
		Value: &Request_Flush{&RequestFlush{}},
	}
}

func ToRequestInfo(req RequestInfo) *Request {
	return &Request{
		Value: &Request_Info{&req},
	}
}

func ToRequestSetOption(req RequestSetOption) *Request {
	return &Request{
		Value: &Request_SetOption{&req},
	}
}

func ToRequestDeliverTx(tx []byte) *Request {
	return &Request{
		Value: &Request_DeliverTx{&RequestDeliverTx{tx}},
	}
}
func ToRequestDeliverTxs(txs [][]byte) *Request {
	return &Request{
		Value: &Request_DeliverTxs{&RequestDeliverTxs{txs}},
	}
}

func ToRequestCheckTx(tx []byte) *Request {
	return &Request{
		Value: &Request_CheckTx{&RequestCheckTx{tx}},
	}
}
func ToRequestCheckTxs(txs [][]byte) *Request {
	return &Request{
		Value: &Request_CheckTxs{&RequestCheckTxs{txs}},
	}
}
func ToRequestCheckTxConcurrency(tx []byte) *Request {
	return &Request{
		Value: &Request_CheckTxConcurrency{&RequestCheckTx{tx}},
	}
}

func ToRequestCommit() *Request {
	return &Request{
		Value: &Request_Commit{&RequestCommit{}},
	}
}

func ToRequestQuery(req RequestQuery) *Request {
	return &Request{
		Value: &Request_Query{&req},
	}
}

func ToRequestQueryEx(req RequestQueryEx) *Request {
	return &Request{
		Value: &Request_QueryEx{&req},
	}
}

func ToRequestInitChain(req RequestInitChain) *Request {
	return &Request{
		Value: &Request_InitChain{&req},
	}
}

func ToRequestBeginBlock(req RequestBeginBlock) *Request {
	return &Request{
		Value: &Request_BeginBlock{&req},
	}
}

func ToRequestEndBlock(req RequestEndBlock) *Request {
	return &Request{
		Value: &Request_EndBlock{&req},
	}
}

func ToRequestCleanData() *Request {
	return &Request{
		Value: &Request_CleanData{&RequestCleanData{}},
	}
}

func ToRequestGetGenesis() *Request {
	return &Request{
		Value: &Request_GetGenesis{&RequestGetGenesis{}},
	}
}

func ToRequestRollback() *Request {
	return &Request{
		Value: &Request_Rollback{&RequestRollback{}},
	}
}

//----------------------------------------

func ToResponseException(errStr string) *Response {
	return &Response{
		Value: &Response_Exception{&ResponseException{errStr}},
	}
}

func ToResponseEcho(message string) *Response {
	return &Response{
		Value: &Response_Echo{&ResponseEcho{message}},
	}
}

func ToResponseFlush() *Response {
	return &Response{
		Value: &Response_Flush{&ResponseFlush{}},
	}
}

func ToResponseInfo(res ResponseInfo) *Response {
	return &Response{
		Value: &Response_Info{&res},
	}
}

func ToResponseSetOption(res ResponseSetOption) *Response {
	return &Response{
		Value: &Response_SetOption{&res},
	}
}

func ToResponseDeliverTx(res ResponseDeliverTx) *Response {
	return &Response{
		Value: &Response_DeliverTx{&res},
	}
}
func ToResponseDeliverTxs(res ResponseDeliverTxs) *Response {
	return &Response{
		Value: &Response_DeliverTxs{&res},
	}
}
func ToResponseCheckTx(res ResponseCheckTx) *Response {
	return &Response{
		Value: &Response_CheckTx{&res},
	}
}
func ToResponseCheckTxs(res ResponseCheckTxs) *Response {
	return &Response{
		Value: &Response_CheckTxs{&res},
	}
}

func ToResponseCommit(res ResponseCommit) *Response {
	return &Response{
		Value: &Response_Commit{&res},
	}
}

func ToResponseQuery(res ResponseQuery) *Response {
	return &Response{
		Value: &Response_Query{&res},
	}
}

func ToResponseQueryEx(res ResponseQueryEx) *Response {
	return &Response{
		Value: &Response_QueryEx{&res},
	}
}

func ToResponseInitChain(res ResponseInitChain) *Response {
	return &Response{
		Value: &Response_InitChain{&res},
	}
}

func ToResponseBeginBlock(res ResponseBeginBlock) *Response {
	return &Response{
		Value: &Response_BeginBlock{&res},
	}
}

func ToResponseEndBlock(res ResponseEndBlock) *Response {
	return &Response{
		Value: &Response_EndBlock{&res},
	}
}

func ToResponseCleanData(res ResponseCleanData) *Response {
	return &Response{
		Value: &Response_CleanData{&res},
	}
}

func ToResponseGetGenesis(res ResponseGetGenesis) *Response {
	return &Response{
		Value: &Response_GetGenesis{&res},
	}
}

func ToResponseRollback(res ResponseRollback) *Response {
	return &Response{
		Value: &Response_Rollback{&res},
	}
}
