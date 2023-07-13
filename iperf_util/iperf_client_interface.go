package iperf_utils

type IperfClientInterface interface {
	Start() error
	Stop() error
	GetNewestMsg() (IperfResult, error)
	GetResult() string
	GetKey() string
	GetReturnFlows() *ReturnFlows
	IsUDP() bool
	IsRunning() bool
}

type Protocol string

const (
	ProtoTcp  Protocol = "TCP"
	ProtoUdp  Protocol = "UDP"
	ProtoNone Protocol = "NONE"
)

func ToProto(proto string) Protocol {
	if proto == "tcp" || proto == "TCP" {
		return ProtoTcp
	}
	if proto == "udp" || proto == "UDP" {
		return ProtoUdp
	}
	return ProtoNone
}
