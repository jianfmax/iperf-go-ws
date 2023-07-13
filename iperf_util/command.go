package iperf_utils

import (
	"bufio"
	"context"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"syscall"
)

// Command 异步执行某个命令
// 输出结果的管道，按照行输出
// 输出程序执行的返回代码。0：正常执行
// error: 开始执行该命令失败
func Command(ctx context.Context, cmd string) (<-chan string, <-chan int, error) {
	result := make(chan string, 100)
	resultCode := make(chan int)
	c := exec.CommandContext(ctx, "bash", "-c", cmd) // mac linux
	stdout, err := c.StdoutPipe()
	if err != nil {
		return result, resultCode, err
	}
	go func() {
		defer func() {
			err := recover()
			if err != nil {
				log.Error("崩溃性错误")
				log.Error(err)
			}
		}()
		reader := bufio.NewReader(stdout)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				readString, err := reader.ReadString('\n')
				if err != nil {
					log.Error(err)
				}
				if err != nil && err.Error() == "read |0: file already closed" {
					close(result)
					return
				}
				if err == nil {
					result <- readString
				}
			}

		}
	}()
	err = c.Start()
	go func() {
		defer func() {
			err := recover()
			if err != nil {
				log.Error("崩溃性错误")
				log.Error(err)
			}
		}()
		if err := c.Wait(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					log.Infof("程序非法退出:%v", status.ExitStatus())
					resultCode <- status.ExitStatus()
				}
			}
		} else {
			resultCode <- 0
		}
		close(resultCode)
	}()
	return result, resultCode, err
}
