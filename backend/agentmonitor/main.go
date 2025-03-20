package main

import (
	"cmd/agentmonitor/data"
	"flag"
	"fmt"
	"time"

	"github.com/go-co-op/gocron"
)

func main() {
	// 定义一个字符串变量，用来接收传入的 host_name 与token参数
	hostName := flag.String("host_name", "", "The hostname for the agent")
	token := flag.String("token", "", "A string of 16 characters")

	// 解析命令行参数
	flag.Parse()

	// 定义服务器端点的URL
	serverURL := "http://192.168.51.28:8080/agent/system_info" // 你的服务器URL

	//创建调度器
	s := gocron.NewScheduler(time.UTC)
	// 每分钟执行一次任务
	s.Every(1).Minute().Do(func() {
		// 收集监控数据
		datas, err := data.CollectMonitorData(*hostName, *token)
		if err != nil {
			fmt.Println("收集数据错误")
		}
		// 发送收集到的数据到服务器
		err = data.SendMonitorData(serverURL, datas)
		if err != nil {
			fmt.Printf("发送数据错误%v", err)
		}
	})
	s.StartBlocking()

}
