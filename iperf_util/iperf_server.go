package iperf_utils

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"sync"
)

type IperfServer struct {
	port         int
	Ctx          context.Context
	close        context.CancelFunc
	codeChan     <-chan int
	locker       sync.Mutex
	ResultStruct chan IperfResult
	AllResult    []string
	started      bool
}

// NewIperfServer 创建一个新的iperf服务端
func NewIperfServer(iperfPath string, port int) (*IperfServer, error) {
	str := fmt.Sprintf("%v -s -p %v --forceflush", iperfPath, port)
	ctx, cancel := context.WithCancel(context.Background())
	resultChan, codeChan, err := Command(ctx, str)
	if err != nil {
		cancel()
		return nil, err
	}
	i := &IperfServer{
		port:         port,
		Ctx:          ctx,
		close:        cancel,
		codeChan:     codeChan,
		ResultStruct: make(chan IperfResult, 100),
		AllResult:    make([]string, 0, 100),
	}
	go func(ctx context.Context, i *IperfServer) {
		defer func() {
			err := recover()
			if err != nil {
				log.Error("崩溃性错误")
				log.Error(err)
			}
		}()
		defer i.Close()
		for {
			select {
			case result, ok := <-resultChan:
				if !ok {
					log.Info("resultChan is closed")
					return
				}
				i.locker.Lock()
				i.AllResult = append(i.AllResult, result)
				log.Errorf("%p -> 获取到的iperf结果为: %v", i, result)
				data, isData := newIperfResultServer(result)
				if isData {
					if i.started && data.Head {
						return
					}
					if !i.started && data.Head {
						i.started = true
					}
					i.ResultStruct <- data
				}
				i.locker.Unlock()
			case code, ok := <-i.codeChan:
				if !ok {
					log.Error("codeChan is closed")
					return
				}
				log.Errorf("iperf server 退出, 代码: %v", code)
				return
			case <-ctx.Done():
				log.Debug("程序退出了")
				return
			}
		}
	}(ctx, i)
	return i, nil
}

func (s *IperfServer) Close() {
	if s.close != nil {
		s.close()
	}
}
