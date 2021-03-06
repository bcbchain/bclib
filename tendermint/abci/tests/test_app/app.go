package main

import (
	"bytes"
	"fmt"

	"github.com/bcbchain/bclib/tendermint/abci/client"
	"github.com/bcbchain/bclib/tendermint/abci/types"
	"github.com/bcbchain/bclib/tendermint/tmlibs/log"
)

func startClient(abciType string) abcicli.Client {
	// Start client
	client, err := abcicli.NewClient("tcp://127.0.0.1:46658", abciType, true)
	if err != nil {
		panic(err.Error())
	}
	logger := log.NewTMLogger("./log", "test_app")
	client.SetLogger(logger.With("module", "abcicli"))
	if err := client.Start(); err != nil {
		panicf("connecting to abci_app: %v", err.Error())
	}

	return client
}

func setOption(client abcicli.Client, key, value string) {
	_, err := client.SetOptionSync(types.RequestSetOption{key, value})
	if err != nil {
		panicf("setting %v=%v: \nerr: %v", key, value, err)
	}
}

func commit(client abcicli.Client, hashExp []byte) {
	res, err := client.CommitSync()
	if err != nil {
		panicf("client error: %v", err)
	}
	if !bytes.Equal(res.GetLastAppHash(), hashExp) {
		panicf("Commit hash was unexpected. Got %X expected %X", res.GetLastAppHash(), hashExp)
	}
}

func deliverTx(client abcicli.Client, txBytes []byte, codeExp uint32, dataExp []byte) {
	res, err := client.DeliverTxSync(txBytes)
	if err != nil {
		panicf("client error: %v", err)
	}
	if res.Code != codeExp {
		panicf("DeliverTx response code was unexpected. Got %v expected %v. Log: %v", res.Code, codeExp, res.Log)
	}
	if !bytes.Equal([]byte(res.Data), dataExp) {
		panicf("DeliverTx response data was unexpected. Got %X expected %X", res.Data, dataExp)
	}
}

/*func checkTx(client abcicli.Client, txBytes []byte, codeExp uint32, dataExp []byte) {
	res, err := client.CheckTxSync(txBytes)
	if err != nil {
		panicf("client error: %v", err)
	}
	if res.IsErr() {
		panicf("checking tx %X: %v\nlog: %v", txBytes, res.Log)
	}
	if res.Code != codeExp {
		panicf("CheckTx response code was unexpected. Got %v expected %v. Log: %v",
			res.Code, codeExp, res.Log)
	}
	if !bytes.Equal(res.Data, dataExp) {
		panicf("CheckTx response data was unexpected. Got %X expected %X",
			res.Data, dataExp)
	}
}*/

func panicf(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}
