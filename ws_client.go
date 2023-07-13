package iperf_go_ws

import (
	iperf_utils "github.com/jianfmax/iperf-go-ws/iperf_util"
)

func IperfClientWS(conn *WsConn, iperfPath string, allTime, port int, proto iperf_utils.Protocol, destinationAddr,
	bandwidthToSend, datagramSize string) error {
	iperfClient, err := iperf_utils.NewIperfClient(iperfPath, allTime, port, proto, destinationAddr, bandwidthToSend, datagramSize)
	if err != nil {
		return err
	}
	for {
		select {
		case <-iperfClient.Ctx.Done():
			return nil
		case data := <-conn.ReadMsgChan():
			if string(data) == "end" {
				iperfClient.Close()
				return nil
			}
		}
	}
}
