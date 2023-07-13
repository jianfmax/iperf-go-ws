package iperf_utils

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"strings"
	"sync"
	"time"
)

// iperfClient iperf的客户端
// 该客户端是一次性的，执行完成后需要退出。该客户端所有执行的结果都存放在该结构体中
type iperfClient struct {
	isServer        bool
	command         string
	Proto           Protocol
	SourceAddr      string
	DestinationAddr string
	Port            int
	BandwidthToSend string
	DatagramSize    string
	AllTimeToSend   int // 单位 秒
	LeftTime        float64
	locker          sync.RWMutex       // 防止竞态的锁
	running         bool               // 该客户端是否正在运行
	closeClient     context.CancelFunc // 提前结束该客户端的函数
	IperfResult     []IperfResult      // 所有解析过的结果
	Result          string             // 所有输出的结果
	Status          string             // 此时获取的信息的状态
}

// NewIperfServer 创建服务端, 因为内容变化不大，没有使用options
// allTime 该程序的所有时间
// port 监听的端口
func NewIperfServerOld(iperfPath string, allTime, port int, proto Protocol, sourceAddr, destinationAddr string) *iperfClient {
	str := fmt.Sprintf("%v -s -p %v --forceflush", iperfPath, port)
	log.Infof("创建Iperf服务端的命令为:%v ", str)
	resultList := make([]IperfResult, 0)
	return &iperfClient{
		isServer:        true,
		command:         str,
		Proto:           proto,
		SourceAddr:      sourceAddr,
		DestinationAddr: destinationAddr,
		Port:            port,
		AllTimeToSend:   allTime,
		LeftTime:        float64(allTime),
		IperfResult:     resultList,
	}
}

// NewIperfClient 创建客户端, 因为内容变化不大，没有使用options
// allTime 该程序的执行时间
// port 监听的端口
// proto 发送的类型
// bandwidthToSend 发送带宽，只有UDP时该参数才起作用，TCP时不起作用。
// datagramSize 每个包的大小，只有UDP时该参数才起作用，TCP时不起作用。
func NewIperfClientOld(iperfPath string, allTime, port int, proto Protocol, sourceAddr, destinationAddr,
	bandwidthToSend, datagramSize string) *iperfClient {
	command := strings.Builder{}
	_, _ = fmt.Fprintf(&command, "%v -c %v -p %v -t %v", iperfPath, destinationAddr, port, allTime)
	if proto == ProtoUdp {
		_, _ = fmt.Fprintf(&command, " -u -b %v -l %v", bandwidthToSend, datagramSize)
	}
	_, _ = fmt.Fprintf(&command, " --forceflush")
	log.Infof("创建Iperf服务端的命令为:%v ", command.String())
	resultList := make([]IperfResult, 0)
	return &iperfClient{
		command:         command.String(),
		Proto:           proto,
		SourceAddr:      sourceAddr,
		DestinationAddr: destinationAddr,
		Port:            port,
		BandwidthToSend: bandwidthToSend,
		DatagramSize:    datagramSize,
		AllTimeToSend:   allTime,
		LeftTime:        float64(allTime),
		IperfResult:     resultList,
	}
}

// Start 开启服务
func (i *iperfClient) Start() error {
	if i.command == "" {
		return errors.New("没有命令可以执行")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(i.AllTimeToSend+20)*time.Second)
	i.closeClient = cancel
	resultChan, codeChan, err := Command(ctx, i.command)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	i.locker.Lock()
	i.running = true
	i.locker.Unlock()
	timeOut := time.NewTimer(time.Duration(2) * time.Second)
	go func(ctx context.Context) {
		defer func() {
			err := recover()
			if err != nil {
				log.Error("崩溃性错误")
				log.Error(err)
			}
		}()
		for {
			select {
			case result, ok := <-resultChan:
				if !ok {
					log.Info("resultChan is closed")
					i.locker.Lock()
					i.Status = "s"
					i.locker.Unlock()
					log.Info("程序退出了")
					return
				}
				i.locker.Lock()
				i.Result += result
				log.Debugf("%p -> 获取到的iperf结果为: %v", i, result)
				data, isData := NewIperfResult(result)
				if isData {
					i.IperfResult = append(i.IperfResult, data)
					//log.Errorf("%p -> 获取到的iperf结果为: %v", i, result)
					log.Debug(data.String())
				}
				i.LeftTime -= data.Interval
				i.Status = "r"
				i.locker.Unlock()
			case code, ok := <-codeChan:
				if !ok {
					log.Error("codeChan is closed")
					return
				}
				i.locker.Lock()
				i.Status = "s"
				i.locker.Unlock()
				log.Warnf("code:%v", code)
				return
			case <-ctx.Done():
				i.locker.Lock()
				i.Status = "s"
				i.locker.Unlock()
				log.Info("程序退出了")
				return
			case <-timeOut.C:
				i.locker.Lock()
				i.Status = "e"
				i.locker.Unlock()
			}
			timeOut.Reset(time.Duration(2) * time.Second)
		}
	}(ctx)
	return nil
}

// Stop 关闭服务
func (i *iperfClient) Stop() error {
	i.locker.Lock()
	defer i.locker.Unlock()
	if i.running {
		i.closeClient()
		i.running = false
	}
	return nil
}

// GetNewestMsg 获取最新的信息
func (i *iperfClient) GetNewestMsg() (IperfResult, error) {
	i.locker.RLock()
	defer i.locker.RUnlock()
	log.Debugf("%p GetNewestMsg -> 获取到的iperf结果数为: %v", i, len(i.IperfResult))
	if len(i.IperfResult) == 0 {
		return IperfResult{}, errors.New("没有任何信息")
	}
	return i.IperfResult[len(i.IperfResult)-1], nil
}

func (i *iperfClient) GetReturnFlows() *ReturnFlows {
	i.locker.RLock()
	defer i.locker.RUnlock()
	bw := strings.TrimSuffix(i.BandwidthToSend, "M")
	r := &ReturnFlows{
		File: fmt.Sprintf("%v_%v_%v_%v", i.Proto, i.Port,
			i.SourceAddr, i.DestinationAddr),
		TimeLeft: fmt.Sprintf("%v", i.LeftTime),
		Bw:       bw,
		Len:      i.DatagramSize,
		Status:   i.Status,
	}
	return r
}

// GetResult 获取所有打印的信息
func (i *iperfClient) GetResult() string {
	i.locker.RLock()
	defer i.locker.RUnlock()
	return i.Result
}

// GetKey 获取该客户端的key
func (i *iperfClient) GetKey() string {
	if i.isServer {
		return fmt.Sprintf("%v_%v_%v_%v_server", i.Proto, i.Port, i.SourceAddr, i.DestinationAddr)
	} else {
		return fmt.Sprintf("%v_%v_%v_%v_client", i.Proto, i.Port, i.SourceAddr, i.DestinationAddr)
	}
}

func (i *iperfClient) IsUDP() bool {
	if i.Proto == ProtoTcp {
		return false
	} else {
		return true
	}
}

func (i *iperfClient) IsRunning() bool {
	return i.running
}

func GetIperfClientKey(proto, port, srcIp, dstIp string) string {
	return fmt.Sprintf("%v_%v_%v_%v_client", proto, port, srcIp, dstIp)
}

func GetIperfServerKey(proto, port, srcIp, dstIp string) string {
	return fmt.Sprintf("%v_%v_%v_%v_server", proto, port, srcIp, dstIp)
}
