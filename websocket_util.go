package iperf_go_ws

import (
	"context"
	"errors"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"sync"
)

// WsConn 封装的基本结构体
type WsConn struct {
	inChan  chan []byte
	outChan chan []byte
	ctx     context.Context
	cancel  context.CancelFunc
	IsClose bool // 通道closeChan是否已经关闭
	Mutex   sync.Mutex
	Conn    *websocket.Conn
}

// InitWebSocket 初始化Websocket
func InitWebSocket(conn *websocket.Conn) *WsConn {
	defer func() {
		err := recover()
		if err != nil {
			log.Error("程序崩溃！")
			log.Error(err)
		}
	}()
	ctx, cancel := context.WithCancel(context.Background())
	ws := &WsConn{
		inChan:  make(chan []byte, 1024),
		outChan: make(chan []byte, 1024),
		ctx:     ctx,
		cancel:  cancel,
		Conn:    conn,
	}
	// 完善必要协程：读取客户端数据协程/发送数据协程
	go ws.ReadMsgLoop()
	go ws.WriteMsgLoop()
	return ws
}

// ReadMsg 读取inChan的数据
func (conn *WsConn) ReadMsg() (data []byte, err error) {
	select {
	case data = <-conn.inChan:
	case <-conn.ctx.Done():
		err = errors.New("connection is closed")
	}
	return
}

func (conn *WsConn) ReadMsgChan() <-chan []byte {
	return conn.inChan
}

// WriteMsg outChan写入数据
func (conn *WsConn) WriteMsg(data []byte) error {
	select {
	case conn.outChan <- data:
		return nil
	case <-conn.ctx.Done():
		return errors.New("connection is closed")
	}
}

// CloseConn 关闭WebSocket连接
func (conn *WsConn) CloseConn() {
	// 关闭closeChan以控制inChan/outChan策略,仅此一次
	conn.Mutex.Lock()
	if !conn.IsClose {
		conn.cancel()
		close(conn.inChan)
		close(conn.outChan)
		conn.IsClose = true
	}
	conn.Mutex.Unlock()
	//关闭WebSocket的连接,Conn.Close()是并发安全可以多次关闭
	_ = conn.Conn.Close()
}

// ReadMsgLoop 读取客户端发送的数据写入到inChan
func (conn *WsConn) ReadMsgLoop() {
	defer func() {
		err := recover()
		if err != nil {
			log.Error("崩溃性错误")
			log.Error(err)
		}
	}()
	for {
		// 确定数据结构
		var (
			data []byte
			err  error
		)
		// 接受数据
		if _, data, err = conn.Conn.ReadMessage(); err != nil {
			conn.CloseConn()
		}
		select {
		case conn.inChan <- data:
			log.Errorf("获取到的数据:%v", string(data))
		case <-conn.ctx.Done():
			return
		default:
			log.Errorf("获取到的数据:%v", string(data))
		}
	}
}

// WriteMsgLoop 读取outChan的数据响应给客户端
func (conn *WsConn) WriteMsgLoop() {
	defer func() {
		err := recover()
		if err != nil {
			log.Error("崩溃性错误")
			log.Error(err)
		}
	}()
	for {
		var (
			data []byte
			err  error
		)
		// 读取数据
		select {
		case data = <-conn.outChan:
			if err = conn.Conn.WriteMessage(1, data); err != nil {
				conn.CloseConn()
			}
		case <-conn.ctx.Done():
			err = errors.New("connection is closed")
		}
	}
}
