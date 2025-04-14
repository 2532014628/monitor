package model

import (
	"backend/server/config"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/lib/pq"
)

var DB *sql.DB
var TDengine *sql.DB

// 连接数据库并创建表
func InitDB() (db *sql.DB, tdengine *sql.DB, err error) { //
	// connStr := "host=192.168.31.251 port=5432 user=postgres password=cCyjKKMyweCer8f3 dbname=monitor sslmode=disable"
	config, _ := config.LoadConfig()
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DB.Host,
		config.DB.Port,
		config.DB.User,
		config.DB.Password,
		config.DB.Name,
	)

	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return DB, nil, err
	}

	//connStr := "root:taosdata@tcp(127.0.0.1:6030)/severmonitor"
	connStr = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		config.TDengine.User,
		config.TDengine.Password,
		config.TDengine.Host,
		config.TDengine.Port,
		config.TDengine.Name,
	)

	TDengine, err = sql.Open("taosSql", connStr)
	if err != nil {
		return DB, TDengine, err
	}

	return DB, TDengine, nil
}

type RequestData struct {
	CPUInfo  []CPUInfo     `json:"cpu_info"`
	HostInfo HostInfo      `json:"host_info"`
	MemInfo  MemoryInfo    `json:"mem_info"`
	ProInfo  []ProcessInfo `json:"pro_info"`
	NetInfo  []NetworkInfo `json:"net_info"`
}

type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

type HostInfo struct {
	ID         int       `json:"id"` // 添加 ID 字段
	Hostname   string    `json:"host_name"`
	OS         string    `json:"os"`
	Platform   string    `json:"platform"`
	KernelArch string    `json:"kernel_arch"`
	CreatedAt  time.Time `json:"host_info_created_at"` // 添加 CreatedAt 字段
	Token      string    `json:"token"`
}

type CPUInfo struct {
	ID        int       `json:"id"` // 添加 ID 字段
	ModelName string    `json:"model_name"`
	CoresNum  int       `json:"cores_num"`
	Percent   float64   `json:"percent"`
	CreatedAt time.Time `json:"cpu_info_created_at"` // 添加 CreatedAt 字段
}

type ProcessInfo struct {
	ID         int       `json:"id"` // 添加 ID 字段
	PID        int       `json:"pid"`
	CPUPercent float64   `json:"cpu_percent"`
	MemPercent float64   `json:"mem_percent"`
	CreatedAt  time.Time `json:"pro_info_created_at"` // 添加 CreatedAt 字段
}

type MemoryInfo struct {
	ID          int       `json:"id"` // 添加 ID 字段
	Total       string    `json:"total"`
	Available   string    `json:"available"`
	Used        string    `json:"used"`
	Free        string    `json:"free"`
	UserPercent float64   `json:"user_percent"`
	CreatedAt   time.Time `json:"mem_info_created_at"` // 添加 CreatedAt 字段
}

// 定义网络信息结构体
type NetworkInfo struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	BytesRecv uint64    `json:"bytes_recv"` // 接收字节数
	BytesSent uint64    `json:"bytes_sent"` // 发送字节数
	CreatedAt time.Time `json:"net_info_created_at"`
}

func InsertHostInfo(hostInfo HostInfo, username string) error {
	var hostInfoID int
	var hostname string
	var exists bool

	// 检查主机记录是否存在
	querySQL := `
    SELECT id, host_name, EXISTS (SELECT 1 FROM host_info WHERE host_name = $1 AND os = $2 AND platform = $3 AND kernel_arch = $4)
    FROM host_info WHERE host_name = $1 AND os = $2 AND platform = $3 AND kernel_arch = $4`

	err := DB.QueryRow(querySQL, hostInfo.Hostname, hostInfo.OS, hostInfo.Platform, hostInfo.KernelArch).Scan(&hostInfoID, &hostname, &exists)
	if err == sql.ErrNoRows {
		fmt.Println("No matching host info found.")
		exists = false
	} else if err != nil {
		fmt.Printf("Failed to query host info: %v\n", err)
		return err
	}

	if exists {
		// 更新已存在的主机记录
		updateSQL := `
        UPDATE host_info
        SET created_at = CURRENT_TIMESTAMP
        WHERE id = $1`
		_, err = DB.Exec(updateSQL, hostInfoID)
		if err != nil {
			fmt.Printf("Failed to update host_info_created_at: %v\n", err)
			return err
		}
		fmt.Printf("Updated existing host_info with ID: %d\n", hostInfoID)
	} else {
		// 插入新的主机记录
		insertSQL := `
        INSERT INTO host_info (host_name, os, platform, kernel_arch, created_at, user_name)
        VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, $5)
        RETURNING id, host_name`
		err = DB.QueryRow(insertSQL, hostInfo.Hostname, hostInfo.OS, hostInfo.Platform, hostInfo.KernelArch, username).Scan(&hostInfoID, &hostname)
		if err != nil {
			fmt.Printf("Failed to insert host_info: %v\n", err)
			return err
		}
		fmt.Printf("Inserted new host_info with ID and Name: %d and %v\n", hostInfoID, hostname)
	}

	return nil
}

func InsertSystemInfo(hostname string, hostInfo HostInfo, cpuInfo []CPUInfo, memoryInfo MemoryInfo, networkInfo []NetworkInfo) error {
	// 检查是否已经存在对应的 system_info 记录
	var exists bool

	// 查询是否存在
	querySQL := `
	SELECT EXISTS (
        SELECT 1
        FROM host_info
        WHERE host_name = $1
		ORDER BY created_at DESC LIMIT 1
    )`

	err := DB.QueryRow(querySQL, hostname).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("InsertSystemInfo : failed to query host_info's existence: %v", err)
	}
	if !exists {
		return fmt.Errorf("InsertSystemInfo : host_info with hostname '%s' does not exist", hostname)
	}

	tableName := fmt.Sprintf("%s_system_info", hostname)
	// 查询在TDengine中该子表是否存在
	querySQL = fmt.Sprintf(`
        SELECT COUNT(*)
        FROM information_schema.ins_tables
        WHERE table_name = '%s'
    `, tableName)
	err = TDengine.QueryRow(querySQL).Scan(&exists)
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
	// 创建新的数据实例
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

	//processData := processInfo
	//processDataJSON, err := json.Marshal(processData)
	//if err != nil {
	//	return fmt.Errorf("InsertSystemInfo : failed to marshal processData: %v", err)
	//}
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
		INSERT INTO %s (created_at, host_name, host_info, cpu_info, memory_info,network_info) 
		VALUES ('%s', '%s', '%s', '%s', '%s', '%s')`, tableName, currentTime, hostname, string(hostDataJSON), string(cpuDataJSON), string(memoryDataJSON), string(networkDataJSON))
	_, err = TDengine.Exec(insertData)
	if err != nil {
		return fmt.Errorf("failed to g data into table for host %s: %w", hostname, err)
	}
	fmt.Println("Data inserted successfully!")
	return nil
}

func InsertHostandToken(hostname string, Token string) error {
	var existingID int
	// 查询是否存在
	querySQL := `
	SELECT id
	FROM hostandtoken
	WHERE host_name = $1`

	err := DB.QueryRow(querySQL, hostname).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to query hostandtoken: %v", err)
	}
	if existingID > 0 {
		// 更新已存在的主机记录
		updateSQL := `
        UPDATE hostandtoken                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   
		SET 
		    token = $1,
		    last_heartbeat = CURRENT_TIMESTAMP
		WHERE host_name = $2`
		_, err = DB.Exec(updateSQL, Token, hostname)
		if err != nil {
			fmt.Printf("Failed to update hostandtoken's token: %v\n", err)
			return err
		}
		fmt.Printf("Updated existing hostandtoken with token: %d\n", Token)
		//fmt.Println("InsertHostandToken : The host_name already exists!")
		return nil
	}

	// 插入新的记录
	fmt.Println("Inserting new host")
	insertSQL := `
	INSERT INTO hostandtoken (host_name, token)
	VALUES ($1, $2) RETURNING token`
	var token string
	err = DB.QueryRow(insertSQL, hostname, Token).Scan(&token)
	if err != nil {
		log.Fatalf("Failed to query host info: %v\n", err)
		return err
	}
	log.Println("Insert successfully")

	return nil
}

func InsertSSHKeys(hostname string, sshkey string) error {
	var existingID int
	//查询在host_info表中是否存在该主机名
	querySQL := `
	SELECT id
	FROM host_info
	WHERE host_name = $1`

	err := DB.QueryRow(querySQL, hostname).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to query host_info: %v", err)
	}
	if existingID == 0 {
		return fmt.Errorf("host_info with hostname '%s' does not exist", hostname)
	}

	// 查询是否在sshkeys表存在该主机名
	querySQL = `
	SELECT id
	FROM ssh_keys
	WHERE host_name = $1`

	err = DB.QueryRow(querySQL, hostname).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to query sshkeys: %v", err)
	}
	if existingID > 0 {
		// 更新已存在的记录
		updateSQL := `
        UPDATE ssh_keys                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   
		SET 
		    sshkey= $1,
		WHERE host_name = $2`
		_, err = DB.Exec(updateSQL, sshkey, hostname)
		if err != nil {
			fmt.Printf("Failed to update sshkeys's sshkey: %v\n", err)
			return err
		}
		fmt.Printf("Updated existing sshkeys with sshkey: %s\n", sshkey)
		return nil
	}

	// 插入新的记录
	fmt.Println("Inserting new sshkey")
	insertSQL := `
	INSERT INTO ssh_keys (host_name, sshkey)
	VALUES ($1, $2) `
	err = DB.QueryRow(insertSQL, hostname, sshkey).Err()
	if err != nil {
		log.Fatalf("Failed to insert sshkeys info: %v\n", err)
		return err
	}
	log.Println("Insert successfully")

	return nil
}

func ReadMemoryInfo(hostname string, from, to string, result map[string]interface{}) error {
	// 使用 TDengine 连接替代 PostgreSQL 连接
	tableName := fmt.Sprintf("%s_system_info", hostname)

	// 构造 TDengine 查询语句
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

// 统一查询函数结构优化点：
// 1. 所有系统信息查询改用 TDengine
// 2. 增加时间格式处理逻辑
// 3. 优化错误提示信息

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
func ReadDB(queryType, from, to string, hostname string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// 查询主机信息
	if queryType == "host" || queryType == "all" {
		row := DB.QueryRow("SELECT id, host_name, os, platform, kernel_arch, created_at FROM host_info WHERE host_name = $1", hostname)
		var id int
		var os, platform, kernelArch string
		var createdAt time.Time
		err := row.Scan(&id, &hostname, &os, &platform, &kernelArch, &createdAt)
		if err != nil {
			if err == sql.ErrNoRows {
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

func DeleteDB(db *sql.DB, host_id int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// 删除CPU信息
	_, err = tx.Exec("DELETE FROM cpu_info WHERE host_id = $1", host_id)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 删除内存信息
	_, err = tx.Exec("DELETE FROM memory_info WHERE host_id = $1", host_id)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 删除进程信息
	_, err = tx.Exec("DELETE FROM process_info WHERE host_id = $1", host_id)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 删除主机信息
	_, err = tx.Exec("DELETE FROM host_info WHERE host_id = $1", host_id)
	if err != nil {
		tx.Rollback()
		return err
	}

	// 提交事务
	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

// 更新主机信息
func UpdateHostInfo(db *sql.DB, host_id int, host_info map[string]string) error {

	//查看该主机的host_id是否存在
	err := db.QueryRow("SELECT id FROM host_info WHERE host_id = ", host_id).Scan(&host_id)
	if err != nil {
		return fmt.Errorf("failed to query host_info table")
	}
	if err == sql.ErrNoRows {
		return fmt.Errorf("no matching host_id found in host_info table")
	}

	_, err = db.Exec(
		"UPDATE host_info SET host_name = $1, os = $2, platform = $3, kernel_arch = $4 WHERE host_id = $6",
		host_info["Hostname"], host_info["OS"], host_info["Platform"], host_info["KernelArch"], host_id,
	)
	if err != nil {
		return err
	}
	return nil
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

// 更新token表
func UpdateToken(db *sql.DB, hostName string, token string, lastHeartBeat time.Time, status string) error {
	//判断hostandtoken表是否存在该hostname
	var existingName string
	err := db.QueryRow("SELECT hos_tname FROM hostandtoken WHERE host_name = ", hostName).Scan(&existingName)
	if err != nil {
		return err
	}
	if err == sql.ErrNoRows {
		return err
	}

	_, err = db.Exec("UPDATE hostandtoken SET token = ?, last_heartbeat = ?, status = ? WHERE host_name = ?", token, lastHeartBeat, status, hostName)
	if err != nil {
		return err
	}
	return nil
}

func UpdateSSHKeys(db *sql.DB, hostName string, sshKeys string) error {
	//判断sshkeys表是否存在该hostname
	var existingID int
	err := db.QueryRow("SELECT id FROM hostandtoken WHERE host_name = ", hostName).Scan(&existingID)
	if err != nil {
		return err
	}
	if err == sql.ErrNoRows {
		return err
	}

	if existingID <= 0 {
		return fmt.Errorf("hostname not found in sshkeys table")
	}

	_, err = db.Exec("UPDATE sshkeys SET ssh_keys = ? WHERE host_name = ?", sshKeys, hostName)
	if err != nil {
		return err
	}
	return nil
}
