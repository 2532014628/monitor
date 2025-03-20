package data

import (
	"bytes"
	"cmd/agentmonitor/monitor"
	"encoding/json"
	"fmt"
	"net/http"
)

// 定义服务器的监控信息结构体
type MonitorData struct {
	CPUInfo     []monitor.CPUInfo     `json:"cpu_info"`
	HostInfo    monitor.HostInfo      `json:"host_info"`
	MemInfo     monitor.MemoryInfo    `json:"mem_info"`
	ProcessInfo []monitor.ProcessInfo `json:"pro_info"`
	NetworkInfo []monitor.NetworkInfo `json:"net_info"`
}

// 收集监控数据
func CollectMonitorData(hostname string, token string) (MonitorData, error) {
	datas := MonitorData{}

	// 获取CPU使用率
	cpudata, err := monitor.GetCpuInfo()
	if err != nil {
		fmt.Printf("获取CPU信息时出错: %v\n", err)
		return datas, err
	}
	datas.CPUInfo = cpudata

	// 获取内存信息
	memdata, err := monitor.GetMemInfo()
	if err != nil {
		fmt.Printf("获取内存信息时出错: %v\n", err)
		return datas, err
	}
	datas.MemInfo = memdata

	// 获取主机信息
	hostdata, err := monitor.GetHostInfo()
	if err != nil {
		fmt.Printf("获取主机信息时出错: %v\n", err)
		return datas, err
	}
	hostdata.Hostname = hostname
	hostdata.Token = token
	datas.HostInfo = hostdata

	// 获取进程信息
	prodata, err := monitor.GetProcess()
	if err != nil {
		fmt.Printf("获取进程信息时出错: %v\n", err)
		return datas, err
	}
	datas.ProcessInfo = prodata
	// 获取进程信息
	netdata, err := monitor.GetNetworkInfo()
	if err != nil {
		fmt.Printf("获取进程信息时出错: %v\n", err)
		return datas, err
	}
	datas.NetworkInfo = netdata

	return datas, nil
}

// 发送监控数据到服务器
func SendMonitorData(url string, data MonitorData) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("数据序列化错误: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("发送数据错误: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("发送数据失败: %s", resp.Status)
	}

	return err
}
