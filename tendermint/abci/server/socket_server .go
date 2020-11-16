package server

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/bcbchain/bclib/tendermint/abci/types"
	cmn "github.com/bcbchain/bclib/tendermint/tmlibs/common"
)

var maxNumberConnections = runtime.NumCPU() * 8

type SocketServer struct {
	cmn.BaseService

	proto    string
	addr     string
	listener net.Listener

	connsMtx   sync.Mutex
	conns      map[int]net.Conn
	nextConnID int

	//appMtx sync.Mutex
	app types.Application
}

func NewSocketServer(protoAddr string, app types.Application) cmn.Service {
	proto, addr := cmn.ProtocolAndAddress(protoAddr)
	s := &SocketServer{
		proto:    proto,
		addr:     addr,
		listener: nil,
		app:      app,
		conns:    make(map[int]net.Conn),
	}
	s.BaseService = *cmn.NewBaseService(nil, "ABCIServer", s)
	return s
}

func (s *SocketServer) OnStart() error {
	if err := s.BaseService.OnStart(); err != nil {
		return err
	}
	ln, err := net.Listen(s.proto, s.addr)
	if err != nil {
		return err
	}
	s.listener = ln
	go s.acceptConnectionsRoutine()
	return nil
}

func (s *SocketServer) OnStop() {
	s.BaseService.OnStop()
	if err := s.listener.Close(); err != nil {
		s.Logger.Error("Error closing listener", "err", err)
	}

	s.connsMtx.Lock()
	defer s.connsMtx.Unlock()
	for id, conn := range s.conns {
		delete(s.conns, id)
		if err := conn.Close(); err != nil {
			s.Logger.Error("Error closing connection", "id", id, "conn", conn, "err", err)
		}
	}
}

func (s *SocketServer) addConn(conn net.Conn) int {
	s.connsMtx.Lock()
	defer s.connsMtx.Unlock()

	connID := s.nextConnID
	s.nextConnID++
	s.conns[connID] = conn

	return connID
}

// deletes conn even if close errs
func (s *SocketServer) rmConn(connID int) error {
	s.connsMtx.Lock()
	defer s.connsMtx.Unlock()

	conn, ok := s.conns[connID]
	if !ok {
		return fmt.Errorf("Connection %d does not exist", connID)
	}

	delete(s.conns, connID)
	return conn.Close()
}

func (s *SocketServer) acceptConnectionsRoutine() {
	var remoteIp string
	var connID int
	dataChan := make(chan bool)
	for {
		// Accept a connection
		s.Logger.Info("Waiting for new connection...")
		conn, err := s.listener.Accept()
		if err != nil {
			if !s.IsRunning() {
				return // Ignore error from listener closing.
			}
			s.Logger.Error("Failed to accept connection: " + err.Error())
			continue
		}

		addr := conn.RemoteAddr().String()
		currentRemoteIp := strings.Split(addr, ":")[0]

		if remoteIp == "" {
			//起go协程，设置select，select一个case接收channel数据，接收到数据后重置计时器，一个case接收超时，超时后重启bcchain
			go s.checkReqTimeOutInfo(dataChan)
			remoteIp = currentRemoteIp
		} else {
			if currentRemoteIp != remoteIp {
				//拒绝连接
				s.Logger.Error("Connection refused because client ip is invalid", "client ip", currentRemoteIp)
				conn.Close()
				continue
			}
		}

		//此处限制，当链接数大于3时，阻止进行连接
		connAmount := len(s.conns)
		if connAmount == 3 {
			s.Logger.Error("There are four connections from same IP.")
			s.killBcchain()
			return
		}

		connID = s.addConn(conn)
		s.Logger.Info("Accepted a new connection", "connID", connID)

		closeConn := make(chan error, 5)              // Push to signal connection closed
		responses := make(chan *types.Response, 1000) // A channel to buffer responses

		// Read requests from conn and deal with them
		go s.handleRequests(closeConn, conn, responses, dataChan)
		// Pull responses from 'responses' and write them to conn.
		go s.handleResponses(closeConn, conn, responses)

		// Wait until signal to close connection
		go s.waitForClose(closeConn, connID)
	}
}

func (s *SocketServer) checkReqTimeOutInfo(dataChan chan bool) {
	//var timer *time.Timer
	timer := time.NewTimer(600 * time.Second)
	for {
		select {
		case <-dataChan: //Timer Reset
			timer.Reset(600 * time.Second)
			s.Logger.Debug("Timer Reset")
		case <-timer.C:
			s.Logger.Warn("no request 600 Seconds, chain is committing suicide")
			s.Logger.Flush()
			s.killBcchain()
		}
	}
}

func (s *SocketServer) waitForClose(closeConn chan error, connID int) {
	err := <-closeConn
	if err == io.EOF {
		s.Logger.Error("Connection was closed by client", "connID", connID)
	} else if err != nil {
		s.Logger.Error("Connection error", "error", err)
	} else {
		// never happens
		s.Logger.Error("Connection was closed.")
	}

	// Close the connection
	if err := s.rmConn(connID); err != nil {
		s.Logger.Error("Error in closing connection", "error", err)
	}
	//杀死bcchain进程
	s.killBcchain()
}

func (s *SocketServer) killBcchain() {
	pid := os.Getpid()
	pstat, err := os.FindProcess(pid)
	if err != nil {
		panic(err.Error())
	}
	err = pstat.Signal(os.Kill) //kill process
	if err != nil {
		panic(err.Error())
	}
}

// Read requests from conn and deal with them
func (s *SocketServer) handleRequests(closeConn chan error, conn net.Conn, responses chan<- *types.Response,
	dataChan chan bool) {

	var deliverTxs []string //用来暂时存储所有区块中deliver的交易
	var leftDeliverNum int  //未处理的deliver交易数量
	var deliverTxsNum int   //该区块中所有deliver交易的数量

	var abciBeginTime time.Time
	//var checkRawChan chan abcicli.ReqRes
	//var reqResAll []abcicli.ReqRes
	//var timer time.Timer

	var bufReader = bufio.NewReader(conn)
	for {
		var req = &types.Request{}
		err := types.ReadMessage(bufReader, req)
		if err != nil {
			if err == io.EOF {
				closeConn <- err
			} else {
				closeConn <- fmt.Errorf("Error reading message: %v", err.Error())
			}
			return
		}
		dataChan <- true
		//s.handleRequestAll(conn, req, responses, &deliverTxs, &leftDeliverNum, &deliverTxsNum, &connID,
		//	checkRawChan, reqResAll, timer)
		s.handleRequest(conn, req, responses, &deliverTxs, &leftDeliverNum, &deliverTxsNum, &abciBeginTime)
	}
}

func (s *SocketServer) handleRequest(conn net.Conn, req *types.Request, responses chan<- *types.Response,
	deliverTxs *[]string, leftDeliverNum *int, deliverTxsNum *int, abciBeginTime *time.Time) {

	switch r := req.Value.(type) {
	case *types.Request_Echo:
		responses <- types.ToResponseEcho(r.Echo.Message)
	case *types.Request_Flush:
		if *leftDeliverNum != 0 { //如果已经有deliver的交易了，就让之前的交易全部进行运算后，在进行flush的传输
			if len(*deliverTxs) != 0 {
				s.Logger.Error("Request_Flush", "The number of transactions Request_Flush sent was", len(*deliverTxs))
				s.app.PutDeliverTxs(*deliverTxs)
				*deliverTxs = make([]string, 0)
			}
			s.HandleDeliverTxsResponses(leftDeliverNum, responses, true)
			responses <- types.ToResponseFlush()
			//} else if s.connType[*connID] == "AppConnMempool" {
			//	responses <- types.ToResponseFlush()
		} else {
			responses <- types.ToResponseFlush()
		}
	case *types.Request_Info:
		addr := conn.RemoteAddr().String()
		spl := strings.Split(addr, ":")
		r.Info.Host = spl[0]
		res := s.app.Info(*r.Info)
		responses <- types.ToResponseInfo(res)
	case *types.Request_SetOption:
		res := s.app.SetOption(*r.SetOption)
		responses <- types.ToResponseSetOption(res)
	case *types.Request_DeliverTx:
		*deliverTxs = append(*deliverTxs, string(r.DeliverTx.Tx))
		*deliverTxsNum++
		*leftDeliverNum++
		if len(*deliverTxs) == maxNumberConnections {
			s.Logger.Error("Request_DeliverTx", "The number of transactions Request_DeliverTx sent was", len(*deliverTxs))
			s.app.PutDeliverTxs(*deliverTxs)
			*deliverTxs = make([]string, 0)
		}
		s.HandleDeliverTxsResponses(leftDeliverNum, responses, false)
	case *types.Request_CheckTx:
		res := s.app.CheckTx(r.CheckTx.Tx)
		responses <- types.ToResponseCheckTx(res)
	case *types.Request_Commit:
		res := s.app.Commit()
		responses <- types.ToResponseCommit(res)
	case *types.Request_Query:
		res := s.app.Query(*r.Query)
		responses <- types.ToResponseQuery(res)
	case *types.Request_QueryEx:
		res := s.app.QueryEx(*r.QueryEx)
		responses <- types.ToResponseQueryEx(res)
	case *types.Request_InitChain:
		res := s.app.InitChain(*r.InitChain)
		responses <- types.ToResponseInitChain(res)
	case *types.Request_BeginBlock:
		s.Logger.Error("Request_BeginBlock")
		*abciBeginTime = time.Now()
		*deliverTxs = make([]string, 0)
		*leftDeliverNum = 0
		*deliverTxsNum = 0
		res := s.app.BeginBlock(*r.BeginBlock)
		responses <- types.ToResponseBeginBlock(res)
	case *types.Request_EndBlock:
		if *leftDeliverNum > 0 || len(*deliverTxs) > 0 {
			s.Logger.Error("Request_EndBlock", "The number of transactions Request_EndBlock sent was", len(*deliverTxs))
			if len(*deliverTxs) != 0 {
				s.app.PutDeliverTxs(*deliverTxs)
				*deliverTxs = make([]string, 0)
			}
			s.HandleDeliverTxsResponses(leftDeliverNum, responses, true)
			s.Logger.Error("Request_EndBlock", "leftDeliverNum", *leftDeliverNum)
		}
		res := s.app.EndBlock(*r.EndBlock)
		responses <- types.ToResponseEndBlock(res)
		abciTime := time.Now().Sub(*abciBeginTime)
		s.Logger.Error("测试结果", "abci的时间", abciTime, "区块中总交易数量为", *deliverTxsNum, "abciTPS", float64(*deliverTxsNum)/abciTime.Seconds(), "区块高度为", r.EndBlock.Height)
	case *types.Request_CleanData:
		res := s.app.CleanData()
		responses <- types.ToResponseCleanData(res)
	case *types.Request_GetGenesis:
		res := s.app.GetGenesis()
		responses <- types.ToResponseGetGenesis(res)
	case *types.Request_Rollback:
		res := s.app.Rollback()
		responses <- types.ToResponseRollback(res)
	default:
		responses <- types.ToResponseException("Unknown request")
	}
}

//func (s *SocketServer) handleRequestAll(conn net.Conn, responses chan<- *types.Response,
//	RequestChan chan *types.Request, connID *int) {
//
//	var deliverTxs []string //用来暂时存储所有区块中deliver的交易
//	var leftDeliverNum int  //未处理的deliver交易数量
//	var deliverTxsNum int   //该区块中所有deliver交易的数量
//
//	var checkRawChan chan abcicli.ReqRes
//	var reqResAll []abcicli.ReqRes
//	var timer time.Timer
//
//	for {
//		select {
//		case req := <-RequestChan:
//			switch req.Value.(type) {
//			case *types.Request_Echo:
//				s.handleQueryRequest(conn, req, responses, &deliverTxs, &leftDeliverNum, &deliverTxsNum, connID)
//			case *types.Request_Flush:
//				switch s.connType[*connID] {
//				case "Consensus":
//					s.handleMempoolRequest(req, responses, checkRawChan)
//				case "Mempool":
//					s.handleConsensusRequest(conn, req, responses, deliverTxs, leftDeliverNum, deliverTxsNum, connID)
//				case "Query":
//					responses <- types.ToResponseFlush()
//				}
//			case *types.Request_Info:
//				s.connsQueryOnce.Do(func() {
//					s.connType[*connID] = "Query"
//				})
//				s.handleQueryRequest(conn, req, responses, deliverTxs, leftDeliverNum, deliverTxsNum, connID)
//			case *types.Request_SetOption:
//				s.handleQueryRequest(conn, req, responses, deliverTxs, leftDeliverNum, deliverTxsNum, connID)
//			case *types.Request_DeliverTx:
//				s.handleConsensusRequest(conn, req, responses, deliverTxs, leftDeliverNum, deliverTxsNum, connID)
//			case *types.Request_CheckTx:
//				s.connsMempoolOnce.Do(func() {
//					s.connType[*connID] = "Mempool"
//					var flag chan []abcicli.ReqRes
//					go s.HandleCheck(checkRawChan, reqResAll, timer, flag)
//					go s.HandleCheckTxsResponses(responses, flag)
//				})
//				s.handleMempoolRequest(req, responses, checkRawChan)
//			case *types.Request_Commit:
//				s.handleConsensusRequest(conn, req, responses, deliverTxs, leftDeliverNum, deliverTxsNum, connID)
//			case *types.Request_Query:
//				s.handleQueryRequest(conn, req, responses, deliverTxs, leftDeliverNum, deliverTxsNum, connID)
//			case *types.Request_QueryEx:
//				s.handleQueryRequest(conn, req, responses, deliverTxs, leftDeliverNum, deliverTxsNum, connID)
//			case *types.Request_InitChain:
//				s.connsConsensusOnce.Do(func() {
//					s.connType[*connID] = "Consensus"
//				})
//				s.handleConsensusRequest(conn, req, responses, deliverTxs, leftDeliverNum, deliverTxsNum, connID)
//			case *types.Request_BeginBlock:
//				s.connsConsensusOnce.Do(func() {
//					s.connType[*connID] = "Consensus"
//				})
//				s.handleConsensusRequest(conn, req, responses, deliverTxs, leftDeliverNum, deliverTxsNum, connID)
//			case *types.Request_EndBlock:
//				s.handleConsensusRequest(conn, req, responses, deliverTxs, leftDeliverNum, deliverTxsNum, connID)
//			case *types.Request_CleanData:
//				s.handleConsensusRequest(conn, req, responses, deliverTxs, leftDeliverNum, deliverTxsNum, connID)
//			case *types.Request_GetGenesis:
//				s.handleQueryRequest(conn, req, responses, deliverTxs, leftDeliverNum, deliverTxsNum, connID)
//			case *types.Request_Rollback:
//				switch s.connType[*connID] {
//				case "Consensus":
//					s.handleMempoolRequest(req, responses, checkRawChan)
//				case "Mempool":
//					s.handleConsensusRequest(conn, req, responses, deliverTxs, leftDeliverNum, deliverTxsNum, connID)
//				}
//			default:
//				responses <- types.ToResponseException("Unknown request")
//			}
//		}
//	}
//
//}

//func (s *SocketServer) handleConsensusRequest(conn net.Conn, req *types.Request, responses chan<- *types.Response,
//	deliverTxs *[]string, leftDeliverNum *int, deliverTxsNum *int, connID *int) {
//
//	switch r := req.Value.(type) {
//	case *types.Request_Flush:
//		if len(*deliverTxs) != 0 {
//			s.Logger.Error("Request_Flush", "The number of transactions Request_Flush sent was", len(*deliverTxs))
//			s.app.PutDeliverTxs(*deliverTxs)
//			*deliverTxs = make([]string, 0)
//		}
//		s.HandleDeliverTxsResponses(leftDeliverNum, responses, true)
//		responses <- types.ToResponseFlush()
//	case *types.Request_DeliverTx:
//		*deliverTxs = append(*deliverTxs, string(r.DeliverTx.Tx))
//		*deliverTxsNum++
//		*leftDeliverNum++
//		if len(*deliverTxs) == maxNumberConnections {
//			s.Logger.Error("Request_DeliverTx", "The number of transactions Request_DeliverTx sent was", len(*deliverTxs))
//			s.app.PutDeliverTxs(*deliverTxs)
//			*deliverTxs = make([]string, 0)
//		}
//		s.HandleDeliverTxsResponses(leftDeliverNum, responses, false)
//	case *types.Request_Commit:
//		res := s.app.Commit()
//		responses <- types.ToResponseCommit(res)
//	case *types.Request_InitChain:
//		res := s.app.InitChain(*r.InitChain)
//		responses <- types.ToResponseInitChain(res)
//	case *types.Request_BeginBlock:
//		*deliverTxs = make([]string, 0)
//		*leftDeliverNum = 0
//		*deliverTxsNum = 0
//		res := s.app.BeginBlock(*r.BeginBlock)
//		responses <- types.ToResponseBeginBlock(res)
//	case *types.Request_EndBlock:
//		if *leftDeliverNum > 0 || len(*deliverTxs) > 0 {
//			s.Logger.Debug("Request_EndBlock", "The number of transactions Request_EndBlock sent was", len(*deliverTxs))
//			if len(*deliverTxs) != 0 {
//				s.app.PutDeliverTxs(*deliverTxs)
//				*deliverTxs = make([]string, 0)
//			}
//			s.HandleDeliverTxsResponses(leftDeliverNum, responses, true)
//			s.Logger.Error("Request_EndBlock", "leftDeliverNum", *leftDeliverNum)
//		}
//		res := s.app.EndBlock(*r.EndBlock)
//		responses <- types.ToResponseEndBlock(res)
//	case *types.Request_CleanData:
//		res := s.app.CleanData()
//		responses <- types.ToResponseCleanData(res)
//	case *types.Request_Rollback:
//		res := s.app.Rollback()
//		responses <- types.ToResponseRollback(res)
//	default:
//		responses <- types.ToResponseException("Unknown request")
//	}
//}
//
//func (s *SocketServer) handleMempoolRequest(req *types.Request, responses chan<- *types.Response,
//	CheckRawChan chan abcicli.ReqRes) {
//
//	switch req.Value.(type) {
//	case *types.Request_Flush:
//		responses <- types.ToResponseFlush()
//	case *types.Request_CheckTx:
//		reqRes := abcicli.NewReqRes(req)
//		CheckRawChan <- *reqRes
//		//res := s.app.CheckTx(r.CheckTx.Tx)
//		//responses <- types.ToResponseCheckTx(res)
//	case *types.Request_Rollback:
//		res := s.app.Rollback()
//		responses <- types.ToResponseRollback(res)
//	default:
//		responses <- types.ToResponseException("Unknown request")
//	}
//}
//
//func (s *SocketServer) handleQueryRequest(conn net.Conn, req *types.Request, responses chan<- *types.Response,
//	deliverTxs *[]string, leftDeliverNum *int, deliverTxsNum *int, connID *int) {
//
//	switch r := req.Value.(type) {
//	case *types.Request_Echo:
//		responses <- types.ToResponseEcho(r.Echo.Message)
//	case *types.Request_Flush:
//		if *leftDeliverNum != 0 { //如果已经有deliver的交易了，就让之前的交易全部进行运算后，在进行flush的传输
//			if len(*deliverTxs) != 0 && s.connType[*connID] == "AppConnConsensus" {
//				s.Logger.Error("Request_Flush", "The number of transactions Request_Flush sent was", len(*deliverTxs))
//				s.app.PutDeliverTxs(*deliverTxs)
//				*deliverTxs = make([]string, 0)
//			}
//			s.HandleDeliverTxsResponses(leftDeliverNum, responses, true)
//			responses <- types.ToResponseFlush()
//		} else if s.connType[*connID] == "AppConnMempool" {
//			responses <- types.ToResponseFlush()
//		} else {
//			responses <- types.ToResponseFlush()
//		}
//	case *types.Request_Info:
//		addr := conn.RemoteAddr().String()
//		spl := strings.Split(addr, ":")
//		r.Info.Host = spl[0]
//		res := s.app.Info(*r.Info)
//		responses <- types.ToResponseInfo(res)
//	case *types.Request_SetOption:
//		res := s.app.SetOption(*r.SetOption)
//		responses <- types.ToResponseSetOption(res)
//	case *types.Request_Query:
//		res := s.app.Query(*r.Query)
//		responses <- types.ToResponseQuery(res)
//	case *types.Request_QueryEx:
//		res := s.app.QueryEx(*r.QueryEx)
//		responses <- types.ToResponseQueryEx(res)
//	case *types.Request_GetGenesis:
//		res := s.app.GetGenesis()
//		responses <- types.ToResponseGetGenesis(res)
//	default:
//		responses <- types.ToResponseException("Unknown request")
//	}
//}

// Pull responses from 'responses' and write them to conn.
func (s *SocketServer) handleResponses(closeConn chan error, conn net.Conn, responses <-chan *types.Response) {
	//var count int
	var bufWriter = bufio.NewWriter(conn)
	for {
		var res = <-responses
		err := types.WriteMessage(res, bufWriter)
		if err != nil {
			closeConn <- fmt.Errorf("Error writing message: %v", err.Error())
			return
		}

		if _, ok := res.Value.(*types.Response_Flush); ok {
			err = bufWriter.Flush()
			if err != nil {
				closeConn <- fmt.Errorf("Error flushing write buffer: %v", err.Error())
				return
			}
		}
	}
}

//// putCheckTxs 将checktx放入，进行并发计算
//func (s *SocketServer) putCheckTxs(reqResAll []abcicli.ReqRes, flag []abcicli.ReqRes) {
//	for _, reqRes := range reqResAll {
//		go func(reqRes abcicli.ReqRes) {
//			res := s.app.CheckTx(reqRes.Request.Value.(*types.Request_CheckTx).CheckTx.Tx)
//			reqRes.SetResponse(types.ToResponseCheckTx(res))
//		}(reqRes)
//	}
//}
//
//func (s *SocketServer) HandleCheck(CheckRawChan chan abcicli.ReqRes, reqResAll []abcicli.ReqRes,
//	timer time.Timer, flag chan []abcicli.ReqRes) {
//	for {
//		select {
//		case reqres := <-CheckRawChan:
//			reqResAll = append(reqResAll, reqres)
//			if len(reqResAll) == runtime.NumCPU()*2 {
//				s.putCheckTxs(reqResAll)
//			}
//		case <-timer.C:
//			s.putCheckTxs(reqResAll)
//		}
//	}
//}
//
//// HandleCheckTxsResponses 将已投入checktx的response返回，严格按照顺序
//func (s *SocketServer) HandleCheckTxsResponses(responses chan<- *types.Response, flag chan []abcicli.ReqRes) {
//	for {
//		select {
//		case <-flag:
//
//		}
//	}
//}

func (s *SocketServer) HandleDeliverTxsResponses(leftNum *int, responses chan<- *types.Response, flag bool) {
	for {
		if res := s.app.GetDeliverTxsResponses(); res != nil {
			//s.Logger.Error("HandleDeliverTxsResponses", "返回的交易的数量为", len(ress))
			//
			*leftNum--
			responses <- types.ToResponseDeliverTx(*res)
		}
		if flag == false || *leftNum == 0 {
			break
		}
	}
}

//func (s *SocketServer) handleReqsent(responses chan<- *types.Response, reqSent *list.List, isChecking *bool,
//	CheckTxChan chan struct{}, leftCheckNum *int) {
//	for {
//		select {
//		case <-CheckTxChan:
//			for {
//				s.Logger.Error("test", "reqSent.Len()", reqSent.Len())
//				if next := reqSent.Front(); next != nil {
//					reqres := next.Value.(*abcicli.ReqRes)
//					if reqres.Response != nil {
//						//s.Logger.Error("handleReqsent", "handleReqsent", reqres.Response.String())
//						responses <- reqres.Response
//						reqSent.Remove(next)
//						break
//					}
//				}
//				time.Sleep(time.Second)
//			}
//		}
//	}
//}
//
//func (s *SocketServer) verifyClientType(connID int, req *types.Request) {
//	switch req.Value.(type) {
//	case *types.Request_CheckTx:
//		s.connType[connID] = "AppConnMempool"
//
//	case *types.Request_Echo:
//		s.connType[connID] = "AppConnQuery"
//	case *types.Request_Query:
//		s.connType[connID] = "AppConnQuery"
//	case *types.Request_QueryEx:
//		s.connType[connID] = "AppConnQuery"
//	case *types.Request_Info:
//		s.connType[connID] = "AppConnQuery"
//	case *types.Request_GetGenesis:
//		s.connType[connID] = "AppConnQuery"
//
//	case *types.Request_InitChain:
//		s.connType[connID] = "AppConnConsensus"
//	case *types.Request_BeginBlock:
//		s.connType[connID] = "AppConnConsensus"
//	case *types.Request_DeliverTx:
//		s.connType[connID] = "AppConnConsensus"
//	case *types.Request_EndBlock:
//		s.connType[connID] = "AppConnConsensus"
//	case *types.Request_Commit:
//		s.connType[connID] = "AppConnConsensus"
//	case *types.Request_CleanData:
//		s.connType[connID] = "AppConnConsensus"
//
//	case *types.Request_SetOption:
//
//	case *types.Request_Flush:
//
//	case *types.Request_Rollback:
//	}
//}
