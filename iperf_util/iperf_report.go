package iperf_utils

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"strconv"
	"strings"
)

type IperfResult struct {
	Interval           float64
	IntervalStart      float64
	IntervalEnd        float64
	Transfer           float64 // 单位：M bit
	Bitrate            float64 // 单位：Mbps
	Jitter             string  // 单位：ms
	LostTotalDatagrams string
	TotalDatagrams     int
	RetryTimes         int
	Cwnd               float64 // 单位：M bit
	Head               bool
}

func NewIperfResult(data string) (IperfResult, bool) {
	result := IperfResult{}
	if len(data) < 5 {
		return result, false
	}
	dataList := strings.Fields(data[5:])
	flag := true
	// udp
	if len(dataList) == 7 {
		// 这里的为获取发送端的信息 udp
		// 形如：[  5]   6.00-7.00   sec   122 KBytes  1000 Kbits/sec  85
		flag = result.setInterval(dataList[0])
		if !flag {
			return result, false
		}
		flag = result.setTransfer(dataList[2], dataList[3])
		if !flag {
			return result, false
		}
		flag = result.setBitrate(dataList[4], dataList[5])
		if !flag {
			return result, false
		}
		result.setTotalDatagrams(dataList[6])
		return result, true
	} else if len(dataList) == 10 {
		// 这里的为获取接收端的信息 udp
		// 形如: [  5]   6.00-7.00   sec   122 KBytes  1000 Kbits/sec  85
		flag = result.setInterval(dataList[0])
		if !flag {
			return result, false
		}
		flag = result.setTransfer(dataList[2], dataList[3])
		if !flag {
			return result, false
		}
		flag = result.setBitrate(dataList[4], dataList[5])
		if !flag {
			return result, false
		}
		result.setJitter(dataList[6], dataList[7])
		result.setLostTotalDatagrams(dataList[8], dataList[9])
		return result, true
	} else if len(dataList) == 6 {
		// tcp 发送和接收是相同的
		flag = result.setInterval(dataList[0])
		if !flag {
			return result, false
		}
		flag = result.setTransfer(dataList[2], dataList[3])
		if !flag {
			return result, false
		}
		flag = result.setBitrate(dataList[4], dataList[5])
		return result, flag
	} else if len(dataList) == 9 {
		// tcp 客户端可能是这个
		// [ ID] Interval           Transfer     Bandwidth       Retr  Cwnd
		// [  4]   0.00-1.00   sec  11.6 MBytes  97.5 Mbits/sec    0    132 KBytes
		flag = result.setInterval(dataList[0])
		if !flag {
			return result, false
		}
		flag = result.setTransfer(dataList[2], dataList[3])
		if !flag {
			return result, false
		}
		flag = result.setBitrate(dataList[4], dataList[5])
		if !flag {
			return result, false
		}
		flag = result.setRetryTimes(dataList[6])
		if !flag {
			return result, false
		}
		flag = result.setCwnd(dataList[7], dataList[8])
		if !flag {
			return result, false
		}
	}
	return result, false
}

func newIperfResultServer(data string) (IperfResult, bool) {
	if strings.HasPrefix(data, "Server listening on") {
		return IperfResult{
			Head: true,
		}, true
	}
	result := IperfResult{}
	if len(data) < 5 {
		return result, false
	}
	dataList := strings.Fields(data[5:])
	flag := true
	switch len(dataList) {
	case 6:
		flag = result.setInterval(dataList[0])
		if !flag {
			return result, false
		}
		flag = result.setTransfer(dataList[2], dataList[3])
		if !flag {
			return result, false
		}
		flag = result.setBitrate(dataList[4], dataList[5])
		return result, flag
	case 8:
		flag = result.setInterval(dataList[0])
		if !flag {
			return result, false
		}
		flag = result.setTransfer(dataList[2], dataList[3])
		if !flag {
			return result, false
		}
		flag = result.setBitrate(dataList[4], dataList[5])
		if !flag {
			return result, false
		}
		flag = result.setCwnd(dataList[6], dataList[7])
		return result, flag
	case 10:
		flag = result.setInterval(dataList[0])
		if !flag {
			return result, false
		}
		flag = result.setTransfer(dataList[2], dataList[3])
		if !flag {
			return result, false
		}
		flag = result.setBitrate(dataList[4], dataList[5])
		if !flag {
			return result, false
		}
		result.setJitter(dataList[6], dataList[7])
		result.setLostTotalDatagrams(dataList[8], dataList[9])
		return result, flag
	default:
		return IperfResult{}, false
	}
}

func (i *IperfResult) setInterval(times string) bool {
	timeFields := strings.Split(times, "-")
	if len(timeFields) != 2 {
		return false
	}
	start, err := strconv.ParseFloat(timeFields[0], 32)
	if err != nil {
		log.Printf("failed to convert start time: %s\n", err)
	}
	end, err := strconv.ParseFloat(timeFields[1], 32)
	i.IntervalStart = start
	i.IntervalEnd = end
	i.Interval = end - start
	return true
}

// setTransfer 获取传输比特数，单位M bit
func (i *IperfResult) setTransfer(data, dimension string) bool {
	transferredStr := fmt.Sprintf("%s%s", data, dimension)
	transferredBytes, ok := getDataBit(transferredStr)
	if !ok {
		return false
	}
	transferredBytes = transferredBytes * 8
	i.Transfer = transferredBytes
	return true
}

// setBitrate 获取传输速率，单位Mbps
func (i *IperfResult) setBitrate(data, dimension string) bool {
	transferredStr := fmt.Sprintf("%s%s", data, dimension)
	bits, ok := getDataBit(transferredStr)
	if !ok {
		return false
	}
	i.Bitrate = bits
	return true
}

// setBitrate 获取传输速率，单位Mbps
func (i *IperfResult) setTotalDatagrams(data string) bool {
	r, err := strconv.Atoi(data)
	if err != nil {
		return false
	}
	i.TotalDatagrams = r
	return true
}

func (i *IperfResult) setJitter(data, dimension string) {
	i.Jitter = fmt.Sprintf("%s%s", data, dimension)
}

func (i *IperfResult) setLostTotalDatagrams(data, dimension string) {
	i.LostTotalDatagrams = fmt.Sprintf("%s %s", data, dimension)
}

func (i *IperfResult) setRetryTimes(data string) bool {
	m, err := strconv.Atoi(data)
	if err != nil {
		return false
	}
	i.RetryTimes = m
	return true
}

func (i *IperfResult) setCwnd(data, dimension string) bool {
	transferredStr := fmt.Sprintf("%s%s", data, dimension)
	transferredBytes, ok := getDataBit(transferredStr)
	if !ok {
		return false
	}
	transferredBytes = transferredBytes * 8
	i.Cwnd = transferredBytes
	return true
}

func getDataBit(data string) (float64, bool) {
	numeric := ""
	units := ""
	for _, c := range data {
		switch {
		case c >= '0' && c <= '9' || c == '.':
			numeric += string(c)
		default:
			units += string(c)
		}
	}
	valueFloat := 0.0
	multiplier := 1.0
	valueFloat, err := strconv.ParseFloat(numeric, 64)
	if err != nil {
		return 0, false
	}
	unitsLower := strings.ToLower(strings.TrimSpace(units))
	switch {
	case strings.HasPrefix(unitsLower, "k"):
		multiplier = 1.0 / 1024.0
	case strings.HasPrefix(unitsLower, "m"):
		multiplier = 1
	case strings.HasPrefix(unitsLower, "g"):
		multiplier = 1024
	case unitsLower == "" || strings.HasPrefix(unitsLower, "b"):
		multiplier = 1.0 / (1024.0 * 1024.0)
	default:
		return 0, false
	}
	value := valueFloat * multiplier
	return value, true
}

type ReturnFlows struct {
	File     string `json:"file,omitempty"`
	TimeLeft string `json:"time_left,omitempty"`
	Bw       string `json:"bw,omitempty"`
	Len      string `json:"len,omitempty"`
	Status   string `json:"status,omitempty"`
}

func (i *IperfResult) ToReturnFlows(client *iperfClient) *ReturnFlows {
	client.locker.RLock()
	defer client.locker.RUnlock()
	r := &ReturnFlows{
		File: fmt.Sprintf("%v_%v_%v_%v", client.Proto, client.Port,
			client.SourceAddr, client.DestinationAddr),
		TimeLeft: fmt.Sprintf("%v", client.LeftTime),
		Bw:       fmt.Sprintf("%v", i.Bitrate),
		Len:      client.DatagramSize,
		Status:   client.Status,
	}
	return r
}

func (i *IperfResult) String() string {
	result := strings.Builder{}
	_, _ = fmt.Fprintf(&result, "Interval: %v; IntervalStart: %v; IntervalEnd:%v; Transfer: %v; "+
		"Bitrate: %v; ",
		i.Interval, i.IntervalStart, i.IntervalEnd, i.Transfer, i.Bitrate)
	if i.Jitter != "" {
		_, _ = fmt.Fprintf(&result, "Jitter: %v; ", i.Jitter)
	}
	if i.LostTotalDatagrams != "" {
		_, _ = fmt.Fprintf(&result, "LostTotalDatagrams: %v; ", i.LostTotalDatagrams)
	}
	if i.TotalDatagrams != 0 {
		_, _ = fmt.Fprintf(&result, "TotalDatagrams: %v; ", i.TotalDatagrams)
	}
	return result.String()
}
