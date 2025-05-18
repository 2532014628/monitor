package model

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"monitor-server/internal/config" // 请根据您的实际项目结构修改这个导入路径
)

var TDengine *sql.DB

// 系统信息相关的结构体定义
type RequestData struct {
	CPUInfo  []CPUInfo
	HostInfo HostInfo
	MemInfo  MemoryInfo
	ProInfo  []ProcessInfo
	NetInfo  []NetworkInfo
}

type HostInfo struct {
	ID         int
	Hostname   string
	OS         string
	Platform   string
	KernelArch string
	CreatedAt  time.Time
	Token      string
}

type CPUInfo struct {
	ID        int
	ModelName string
	CoresNum  int
	Percent   float64
	CreatedAt time.Time
}

type ProcessInfo struct {
	ID         int
	PID        int
	CPUPercent float64
	MemPercent float64
	CreatedAt  time.Time
}

type MemoryInfo struct {
	ID          int
	Total       string
	Available   string
	Used        string
	Free        string
	UserPercent float64
	CreatedAt   time.Time
}

type NetworkInfo struct {
	ID        int
	Name      string
	BytesRecv uint64
	BytesSent uint64
	CreatedAt time.Time
}

func InitDB() (tdengine *sql.DB, err error) { //
	// connStr := "host=192.168.31.251 port=5432 user=postgres password=cCyjKKMyweCer8f3 dbname=monitor sslmode=disable"
	config, _ := config.LoadConfig()

	//connStr := "root:taosdata@tcp(127.0.0.1:6030)/severmonitor"
	connStr := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		config.TDengine.User,
		config.TDengine.Password,
		config.TDengine.Host,
		config.TDengine.Port,
		config.TDengine.Name,
	)

	TDengine, err = sql.Open("taosSql", connStr)
	if err != nil {
		return TDengine, err
	}

	return TDengine, nil
}

// 插入系统信息
func InsertSystemInfo(hostname string, hostInfo HostInfo, cpuInfo []CPUInfo, memoryInfo MemoryInfo, networkInfo []NetworkInfo) error {
	// 检查system_info对应子表是否存在
	var exists bool

	tableName := fmt.Sprintf("%s_system_info", hostname)
	// 查询在TDengine中该子表是否存在
	querySQL := fmt.Sprintf(`
        SELECT COUNT(*)
        FROM information_schema.ins_tables
        WHERE table_name = '%s'
    `, tableName)
	err := TDengine.QueryRow(querySQL).Scan(&exists)
	if err != nil {
		return fmt.Errorf("InsertSystemInfo : failed to query table's existence: %v", err)
	}

	// 获取当前时间并格式化
	currentTime := time.Now().Format("2006-01-02 15:04:05") // 格式化时间为TDengine接受的格式

	// 创建新的数据实例
	hostData := hostInfo
	hostDataJSON, err := json.Marshal(hostData)
	if err != nil {
		return fmt.Errorf("InsertSystemInfo : failed to marshal hostData: %v", err)
	}
	cpuData := cpuInfo
	cpuDataJSON, err := json.Marshal(cpuData)
	if err != nil {
		return fmt.Errorf("InsertSystemInfo : failed to marshal cpuData: %v", err)
	}
	memoryData := memoryInfo
	memoryDataJSON, err := json.Marshal(memoryData)
	if err != nil {
		return fmt.Errorf("InsertSystemInfo : failed to marshal memoryData: %v", err)
	}
	networkData := networkInfo
	networkDataJSON, err := json.Marshal(networkData)
	if err != nil {
		return fmt.Errorf("InsertSystemInfo : failed to marshal networkData: %v", err)
	}

	// 如果子表不存在，则创建子表
	if !exists {
		createTable := fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s USING system_info TAGS ('%s')
		`, tableName, hostname)
		if _, err = TDengine.Exec(createTable); err != nil {
			return fmt.Errorf("failed to create table for host %s: %w", hostname, err)
		}
	}

	// 将新数据插入到TDengine中
	insertData := fmt.Sprintf(`
		INSERT INTO %s (created_at, host_name, host_info, cpu_info, memory_info, network_info) 
		VALUES ('%s', '%s', '%s', '%s', '%s', '%s')`, tableName, currentTime, hostname, string(hostDataJSON), string(cpuDataJSON), string(memoryDataJSON), string(networkDataJSON))
	_, err = TDengine.Exec(insertData)
	if err != nil {
		return fmt.Errorf("failed to insert data into table for host %s: %w", hostname, err)
	}
	fmt.Println("Data inserted successfully!")
	return nil
}

// 读取内存信息
func ReadMemoryInfo(hostname string, from, to string, result map[string]interface{}) error {
	tableName := fmt.Sprintf("%s_system_info", hostname)

	querySQL := fmt.Sprintf(`
        SELECT host_info, cpu_info, memory_info, process_info, network_info 
        FROM %s 
        WHERE created_at >= '%s' AND created_at <= '%s'`,
		tableName, from, to)

	rows, err := TDengine.Query(querySQL)
	if err != nil {
		return fmt.Errorf("查询内存信息时发生错误: %v", err)
	}
	defer rows.Close()

	var memoryData []map[string]interface{}

	for rows.Next() {
		var (
			hostInfoJSON []byte
			cpuInfoJSON  []byte
			memInfoJSON  []byte
			processJSON  []byte
			networkJSON  []byte
		)

		if err := rows.Scan(&hostInfoJSON, &cpuInfoJSON, &memInfoJSON, &processJSON, &networkJSON); err != nil {
			return fmt.Errorf("扫描记录时发生错误: %v", err)
		}

		// 解析 memory_info 字段
		var memInfo MemoryInfo
		if err := json.Unmarshal(memInfoJSON, &memInfo); err != nil {
			return fmt.Errorf("解析内存信息失败: %v", err)
		}

		memoryData = append(memoryData, map[string]interface{}{
			"id":                  memInfo.ID,
			"total":               memInfo.Total,
			"available":           memInfo.Available,
			"used":                memInfo.Used,
			"free":                memInfo.Free,
			"user_percent":        memInfo.UserPercent,
			"mem_info_created_at": memInfo.CreatedAt,
		})
	}

	result["memory"] = memoryData
	return nil
}

// 读取CPU信息
func ReadCPUInfo(hostname string, from, to string, result map[string]interface{}) error {
	tableName := fmt.Sprintf("%s_system_info", hostname)
	querySQL := fmt.Sprintf(`
        SELECT cpu_info 
        FROM %s 
        WHERE created_at >= '%s' AND created_at <= '%s'`,
		tableName, from, to)

	rows, err := TDengine.Query(querySQL)
	if err != nil {
		return fmt.Errorf("CPU信息查询失败: %v", err)
	}
	defer rows.Close()

	var cpuData []map[string]interface{}
	for rows.Next() {
		var cpuInfoJSON []byte
		if err := rows.Scan(&cpuInfoJSON); err != nil {
			return fmt.Errorf("CPU数据扫描失败: %v", err)
		}

		var cpuDataObj []CPUInfo
		if err := json.Unmarshal(cpuInfoJSON, &cpuDataObj); err != nil {
			return fmt.Errorf("CPU数据解析失败: %v", err)
		}
		for _, cpu := range cpuDataObj {
			cpuData = append(cpuData, map[string]interface{}{
				"id":                  cpu.ID,
				"cores_num":           cpu.CoresNum,
				"model_name":          cpu.ModelName,
				"percent":             cpu.Percent,
				"cpu_info_created_at": cpu.CreatedAt,
			})
		}
	}
	result["cpu"] = cpuData
	return nil
}

// 读取进程信息
func ReadProcessInfo(hostname string, from, to string, result map[string]interface{}) error {
	tableName := fmt.Sprintf("%s_system_info", hostname)
	querySQL := fmt.Sprintf(`
        SELECT process_info 
        FROM %s 
        WHERE created_at >= '%s' AND created_at <= '%s'`,
		tableName, from, to)

	rows, err := TDengine.Query(querySQL)
	if err != nil {
		return fmt.Errorf("进程信息查询失败: %v", err)
	}
	defer rows.Close()

	var processData []map[string]interface{}
	for rows.Next() {
		var processJSON []byte
		if err := rows.Scan(&processJSON); err != nil {
			return fmt.Errorf("进程数据扫描失败: %v", err)
		}

		var processDataObj ProcessInfo
		if err := json.Unmarshal(processJSON, &processDataObj); err != nil {
			return fmt.Errorf("进程数据解析失败: %v", err)
		}

		processData = append(processData, map[string]interface{}{
			"id":                  processDataObj.ID,
			"pid":                 processDataObj.PID,
			"cpu_percent":         processDataObj.CPUPercent,
			"mem_percent":         processDataObj.MemPercent,
			"pro_info_created_at": processDataObj.CreatedAt,
		})
	}

	result["process"] = processData
	return nil
}

// 读取网络信息
func ReadNetInfo(hostname string, from, to string, result map[string]interface{}) error {
	tableName := fmt.Sprintf("%s_system_info", hostname)
	querySQL := fmt.Sprintf(`
        SELECT network_info 
        FROM %s 
        WHERE created_at >= '%s' AND created_at <= '%s'`,
		tableName, from, to)

	rows, err := TDengine.Query(querySQL)
	if err != nil {
		return fmt.Errorf("网络信息查询失败: %v", err)
	}
	defer rows.Close()

	var netData []map[string]interface{}
	for rows.Next() {
		var networkJSON []byte
		if err := rows.Scan(&networkJSON); err != nil {
			return fmt.Errorf("网络数据扫描失败: %v", err)
		}

		var netDataObj []NetworkInfo
		if err := json.Unmarshal(networkJSON, &netDataObj); err != nil {
			return fmt.Errorf("网络数据解析失败: %v", err)
		}

		for _, net := range netDataObj {
			netData = append(netData, map[string]interface{}{
				"id":                  net.ID,
				"name":                net.Name,
				"bytes_sent":          net.BytesSent,
				"bytes_recv":          net.BytesRecv,
				"net_info_created_at": net.CreatedAt,
			})
		}
	}

	result["net"] = netData
	return nil
}

// 读取最后一条系统信息
func ReadLastSystemInfo(hostname string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	tableName := fmt.Sprintf("%s_system_info", hostname)
	querySQL := fmt.Sprintf(`
        SELECT host_info, cpu_info, memory_info, process_info, network_info 
        FROM %s 
        ORDER BY created_at DESC 
        LIMIT 1`, tableName)

	rows, err := TDengine.Query(querySQL)
	if err != nil {
		return nil, fmt.Errorf("查询系统信息时发生错误: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("未找到指定主机的系统信息")
	}

	var (
		hostInfoJSON []byte
		cpuInfoJSON  []byte
		memInfoJSON  []byte
		processJSON  []byte
		networkJSON  []byte
	)

	if err := rows.Scan(&hostInfoJSON, &cpuInfoJSON, &memInfoJSON, &processJSON, &networkJSON); err != nil {
		return nil, fmt.Errorf("扫描记录时发生错误: %v", err)
	}

	// 解析 host_info 字段
	var hostInfo HostInfo
	if err := json.Unmarshal(hostInfoJSON, &hostInfo); err != nil {
		return nil, fmt.Errorf("解析主机信息失败: %v", err)
	}
	result["host"] = map[string]interface{}{
		"id":                   hostInfo.ID,
		"host_name":            hostInfo.Hostname,
		"os":                   hostInfo.OS,
		"platform":             hostInfo.Platform,
		"kernel_arch":          hostInfo.KernelArch,
		"host_info_created_at": hostInfo.CreatedAt,
	}

	// 解析 cpu_info 字段
	var cpuDataObj []CPUInfo
	if err := json.Unmarshal(cpuInfoJSON, &cpuDataObj); err != nil {
		return nil, fmt.Errorf("解析 CPU 信息失败: %v", err)
	}
	var cpuData []map[string]interface{}
	for _, cpu := range cpuDataObj {
		cpuData = append(cpuData, map[string]interface{}{
			"id":                  cpu.ID,
			"cores_num":           cpu.CoresNum,
			"model_name":          cpu.ModelName,
			"percent":             cpu.Percent,
			"cpu_info_created_at": cpu.CreatedAt,
		})
	}
	result["cpu"] = cpuData

	// 解析 memory_info 字段
	var memInfo MemoryInfo
	if err := json.Unmarshal(memInfoJSON, &memInfo); err != nil {
		return nil, fmt.Errorf("解析内存信息失败: %v", err)
	}
	result["memory"] = map[string]interface{}{
		"id":                  memInfo.ID,
		"total":               memInfo.Total,
		"available":           memInfo.Available,
		"used":                memInfo.Used,
		"free":                memInfo.Free,
		"user_percent":        memInfo.UserPercent,
		"mem_info_created_at": memInfo.CreatedAt,
	}

	// 解析 process_info 字段
	var processDataObj []ProcessInfo
	if err := json.Unmarshal(processJSON, &processDataObj); err != nil {
		return nil, fmt.Errorf("解析进程信息失败: %v", err)
	}
	var processData []map[string]interface{}
	for _, proc := range processDataObj {
		processData = append(processData, map[string]interface{}{
			"id":                  proc.ID,
			"pid":                 proc.PID,
			"cpu_percent":         proc.CPUPercent,
			"mem_percent":         proc.MemPercent,
			"pro_info_created_at": proc.CreatedAt,
		})
	}
	result["process"] = processData

	// 解析 network_info 字段
	var netDataObj []NetworkInfo
	if err := json.Unmarshal(networkJSON, &netDataObj); err != nil {
		return nil, fmt.Errorf("解析网络信息失败: %v", err)
	}
	var netData []map[string]interface{}
	for _, net := range netDataObj {
		netData = append(netData, map[string]interface{}{
			"id":                  net.ID,
			"name":                net.Name,
			"bytes_sent":          net.BytesSent,
			"bytes_recv":          net.BytesRecv,
			"net_info_created_at": net.CreatedAt,
		})
	}
	result["net"] = netData

	return result, nil
}

// 更新系统信息
func UpdateSystemInfo(hostName string, hostInfo HostInfo, cpuInfo []CPUInfo, memoryInfo MemoryInfo, processInfo []ProcessInfo, networkInfo []NetworkInfo) error {
	// 查询system_info对应子表否存在
	var exists bool

	tableName := fmt.Sprintf("%s_system_info", hostName)
	// 查询在TDengine中该子表是否存在
	querySQL := fmt.Sprintf(`
        SELECT COUNT(*)
        FROM information_schema.tables
        WHERE table_name = '%s'
    `, tableName)
	err := TDengine.QueryRow(querySQL).Scan(&exists)
	if err != nil {
		return fmt.Errorf("InsertSystemInfo : failed to query table's existence: %v", err)
	}
	if err == sql.ErrNoRows {
		return fmt.Errorf("no matching table found in TDengine")
	}

	if !exists {
		return fmt.Errorf("InsertSystemInfo : table does not exist")
	}

	// 获取当前时间并格式化
	currentTime := time.Now().Format("2006-01-02 15:04:05") // 格式化时间为TDengine接受的格式

	// 创建新的数据实例
	hostData := hostInfo
	hostDataJSON, err := json.Marshal(hostData)
	if err != nil {
		return fmt.Errorf("InsertSystemInfo : failed to marshal hostData: %v", err)
	}
	cpuData := cpuInfo
	cpuDataJSON, err := json.Marshal(cpuData)
	if err != nil {
		return fmt.Errorf("InsertSystemInfo : failed to marshal cpuData: %v", err)
	}
	memoryData := memoryInfo
	memoryDataJSON, err := json.Marshal(memoryData)
	if err != nil {
		return fmt.Errorf("InsertSystemInfo : failed to marshal memoryData: %v", err)
	}
	processData := processInfo
	processDataJSON, err := json.Marshal(processData)
	if err != nil {
		return fmt.Errorf("InsertSystemInfo : failed to marshal processData: %v", err)
	}
	networkData := networkInfo
	networkDataJSON, err := json.Marshal(networkData)
	if err != nil {
		return fmt.Errorf("InsertSystemInfo : failed to marshal networkData: %v", err)
	}

	// 将新数据插入到TDengine中
	insertData := fmt.Sprintf(`
		INSERT INTO %s (created_at, host_name, host_info, cpu_info, memory_info, process_info, network_info) 
		VALUES ('%s', '%s', '%s', '%s', '%s', '%s', '%s')
	`, tableName, currentTime, hostName, string(hostDataJSON), string(cpuDataJSON), string(memoryDataJSON), string(processDataJSON), string(networkDataJSON))
	_, err = TDengine.Exec(insertData)
	if err != nil {
		return fmt.Errorf("failed to insert data into table for host %s: %w", hostName, err)
	}

	return nil
}
func ReadDB(queryType, from, to string, hostname string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// 查询主机信息
	if queryType == "host" || queryType == "all" {
		row := TDengine.QueryRow("SELECT id, host_name, os, platform, kernel_arch, created_at FROM host_info WHERE host_name = $1", hostname)
		var id int
		var os, platform, kernelArch string
		var createdAt time.Time
		err := row.Scan(&id, &hostname, &os, &platform, &kernelArch, &createdAt)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("未找到指定的主机记录")
			}
			return nil, fmt.Errorf("查询主机信息时发生错误: %v", err)
		}
		result["host"] = map[string]interface{}{
			"id":                   id,
			"host_name":            hostname,
			"os":                   os,
			"platform":             platform,
			"kernel_arch":          kernelArch,
			"host_info_created_at": createdAt,
		}
	}

	// 查询内存信息
	if queryType == "memory" || queryType == "all" {
		err := ReadMemoryInfo(hostname, from, to, result)
		if err != nil {
			return nil, err
		}
	}
	// 查询网卡信息
	if queryType == "net" || queryType == "all" {
		err := ReadNetInfo(hostname, from, to, result)
		if err != nil {
			return nil, err
		}
	}
	// 查询 CPU 信息
	if queryType == "cpu" || queryType == "all" {
		err := ReadCPUInfo(hostname, from, to, result)
		if err != nil {
			return nil, err
		}
	}

	//// 查询进程信息
	//if queryType == "process" || queryType == "all" {
	//	err := ReadProcessInfo(hostname, from, to, result)
	//	if err != nil {
	//		return nil, err
	//	}
	//}

	return result, nil
}
