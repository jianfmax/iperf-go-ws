package iperf_utils

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"strings"
	"sync"
)

type IperfClient struct {
	port     int
	Ctx      context.Context
	close    context.CancelFunc
	codeChan <-chan int
	locker   sync.Mutex
	started  bool
}

// NewIperfClient 创建一个新的iperf客户端
func NewIperfClient(iperfPath string, allTime, port int, proto Protocol, destinationAddr,
	bandwidthToSend, datagramSize string) (*IperfClient, error) {
	command := strings.Builder{}
	_, _ = fmt.Fprintf(&command, "%v -c %v -p %v -t %v", iperfPath, destinationAddr, port, allTime)
	if proto == ProtoUdp {
		_, _ = fmt.Fprintf(&command, " -u -b %v -l %v", bandwidthToSend, datagramSize)
	}
	_, _ = fmt.Fprintf(&command, " --forceflush")
	log.Infof("创建Iperf服务端的命令为:%v ", command.String())
	ctx, cancel := context.WithCancel(context.Background())
	resultChan, codeChan, err := Command(ctx, command.String())
	if err != nil {
		cancel()
		return nil, err
	}
	i := &IperfClient{
		port:     port,
		Ctx:      ctx,
		close:    cancel,
		codeChan: codeChan,
	}
	go func(ctx context.Context, i *IperfClient) {
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
			case _, ok := <-resultChan:
				if !ok {
					log.Info("resultChan is closed")
					return
				}
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

func (s *IperfClient) Close() {
	if s.close != nil {
		s.close()
	}
}
