package iperf_go_ws

import (
	"encoding/json"
	iperf_utils "github.com/jianfmax/iperf-go-ws/iperf_util"
	log "github.com/sirupsen/logrus"
	"time"
)

func IperfServerWS(conn *WsConn, iperfPath string, port int) error {
	iperfServer, err := iperf_utils.NewIperfServer(iperfPath, port)
	if err != nil {
		return err
	}
	for {
		select {
		case data := <-iperfServer.ResultStruct:
			j, err := json.Marshal(map[string]iperf_utils.IperfResult{
				"result": data,
			})
			if err != nil {
				log.Error(err)
			}
			err = conn.WriteMsg(j)
			if err != nil {
				log.Errorf("写入数据错误：%v", err)
			}
		case <-iperfServer.Ctx.Done():
			j := make([]byte, 0)
			var err error
			if len(iperfServer.AllResult) == 0 {
				j, err = json.Marshal(map[string]string{
					"err": "不存在结果",
				})
				if err != nil {
					log.Error(err)
				}
			} else {
				j, err = json.Marshal(map[string][]string{
					"allResult": iperfServer.AllResult,
				})
				if err != nil {
					log.Error(err)
				}
			}
			err = conn.WriteMsg(j)
			if err != nil {
				log.Errorf("写入数据错误：%v", err)
			}
			time.Sleep(2 * time.Second)
			//conn.Conn.
			return nil
		case data := <-conn.ReadMsgChan():
			if string(data) == "end" {
				iperfServer.Close()
				return nil
			}
		}
	}
}
