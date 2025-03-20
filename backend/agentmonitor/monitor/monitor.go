package monitor

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	"github.com/shirou/gopsutil/process"
)

const (
	NUM_KB  = 1000.0000
	NUM_MIB = 1000000.0000
	NUM_GB  = 1000000000.0000
)

type MemoryInfo struct {
	ID          int       `json:"id"`
	Total       string    `json:"total"`
	Available   string    `json:"available"`
	Used        string    `json:"used"`
	Free        string    `json:"free"`
	UserPercent float64   `json:"user_percent"`
	CreatedAt   time.Time `json:"mem_info_created_at"`
}

// 获取内存信息
func GetMemInfo() (MemoryInfo, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return MemoryInfo{}, fmt.Errorf("获取内存信息失败: %v", err)
	}

	total := HanderUnit(v.Total, NUM_GB, "G")
	available := HanderUnit(v.Available, NUM_GB, "G")
	used := HanderUnit(v.Used, NUM_GB, "G")
	free := HanderUnit(v.Free, NUM_GB, "G")
	userPercent, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", v.UsedPercent), 64)

	return MemoryInfo{
		Total:       total,
		Available:   available,
		Used:        used,
		Free:        free,
		UserPercent: userPercent,
		CreatedAt:   time.Now(),
	}, nil
}

type CPUInfo struct {
	ID        int       `json:"id"`
	ModelName string    `json:"model_name"`
	CoresNum  int       `json:"cores_num"`
	Percent   float64   `json:"percent"`
	CreatedAt time.Time `json:"cpu_info_created_at"`
}

// 获取CPU信息
func GetCpuInfo() ([]CPUInfo, error) {
	cpuInfos := []CPUInfo{}
	percent, err := cpu.Percent(time.Second*14, false)
	if err != nil {
		return nil, fmt.Errorf("获取CPU使用率失败: %v", err)
	}

	infos, err := cpu.Info()
	if err != nil {
		return nil, fmt.Errorf("获取CPU信息失败: %v", err)
	}

	cpuPercent := percent[0]
	for _, ci := range infos {
		cpuInfo := CPUInfo{
			ModelName: ci.ModelName,
			CoresNum:  int(ci.Cores),
			Percent:   cpuPercent,
			CreatedAt: time.Now(),
		}
		cpuInfos = append(cpuInfos, cpuInfo)
	}
	return cpuInfos, nil
}

type HostInfo struct {
	ID         int       `json:"id"`
	Hostname   string    `json:"hostname"`
	OS         string    `json:"os"`
	Platform   string    `json:"platform"`
	KernelArch string    `json:"kernel_arch"`
	CreatedAt  time.Time `json:"host_info_created_at"`
	Token      string    `json:"token"`
}

// 获取主机信息
func GetHostInfo() (HostInfo, error) {
	hInfo, err := host.Info()
	if err != nil {
		return HostInfo{}, fmt.Errorf("获取主机信息失败: %v", err)
	}

	return HostInfo{
		Hostname:   hInfo.Hostname,
		OS:         hInfo.OS,
		Platform:   hInfo.Platform + "-" + hInfo.PlatformVersion + " " + hInfo.PlatformFamily,
		KernelArch: hInfo.KernelArch,
		CreatedAt:  time.Now(),
	}, nil
}

type ProcessInfo struct {
	ID         int       `json:"id"`
	PID        int       `json:"pid"`
	CPUPercent float64   `json:"cpu_percent"`
	MemPercent float32   `json:"mem_percent"`
	Cmdline    string    `json:"cmdline"`
	CreatedAt  time.Time `json:"pro_info_created_at"`
}

// 获取进程信息
func GetProcess() ([]ProcessInfo, error) {
	processes, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("获取进程列表失败: %v", err)
	}

	var processInfos []ProcessInfo

	for _, p := range processes {
		cpuPercent, err := p.CPUPercent()
		if err != nil {
			continue
		}

		memPercent, err := p.MemoryPercent()
		if err != nil {
			continue
		}

		cmdline, err := p.Cmdline()
		if err != nil {
			continue
		}

		processInfos = append(processInfos, ProcessInfo{
			PID:        int(p.Pid),
			CPUPercent: cpuPercent,
			MemPercent: memPercent,
			Cmdline:    cmdline,
			CreatedAt:  time.Now(),
		})
	}

	sort.Slice(processInfos, func(i, j int) bool {
		return processInfos[i].CPUPercent > processInfos[j].CPUPercent
	})
	return processInfos, nil
}

// 定义网络信息结构体
type NetworkInfo struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	BytesRecv uint64    `json:"bytes_recv"` // 接收字节数
	BytesSent uint64    `json:"bytes_sent"` // 发送字节数
	CreatedAt time.Time `json:"net_info_created_at"`
}

// 获取网卡信息
func GetNetworkInfo() ([]NetworkInfo, error) {
	netIO, err := net.IOCounters(true) // true 获取每个网卡的统计信息
	if err != nil {
		return nil, fmt.Errorf("获取网络信息失败: %v", err)
	}

	var networkInfos []NetworkInfo
	for _, io := range netIO {
		networkInfo := NetworkInfo{
			Name:      io.Name,
			BytesRecv: io.BytesRecv,
			BytesSent: io.BytesSent,
			CreatedAt: time.Now(),
		}
		networkInfos = append(networkInfos, networkInfo)
	}

	return networkInfos, nil
}
func HanderUnit(num uint64, numtype float64, typename string) (newnum string) {

	f := fmt.Sprintf("%.2f", float64(num)/numtype)
	return f + typename
}
