package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"monitor-server/internal/config"
	"monitor-server/internal/model"
	pb "monitor-server/proto"

	_ "github.com/taosdata/driver-go/v3/taosSql"
)

type TDengineRepository struct {
	db *sql.DB
}

func NewTDengineRepository(cfg *config.Config) (*TDengineRepository, error) {
	connStr := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		cfg.TDengine.User,
		cfg.TDengine.Password,
		cfg.TDengine.Host,
		cfg.TDengine.Port,
		cfg.TDengine.Name,
	)

	db, err := sql.Open("taosSql", connStr)
	if err != nil {
		return nil, fmt.Errorf("连接TDengine失败: %v", err)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("测试TDengine连接失败: %v", err)
	}

	return &TDengineRepository{db: db}, nil
}

// GetSystemInfo 获取系统信息
func (r *TDengineRepository) GetSystemInfo(ctx context.Context, hostname, from, to string) (*pb.SystemInfo, error) {
	tableName := fmt.Sprintf("%s_system_info", hostname)
	querySQL := fmt.Sprintf(`
		SELECT host_info, cpu_info, memory_info, network_info
		FROM %s
		WHERE created_at >= '%s' AND created_at <= '%s'
	`, tableName, from, to)

	rows, err := r.db.Query(querySQL)
	if err != nil {
		return nil, fmt.Errorf("查询系统信息失败: %v", err)
	}
	defer rows.Close()

	var info pb.SystemInfo
	if rows.Next() {
		var (
			hostInfoJSON    []byte
			cpuInfoJSON     []byte
			memoryInfoJSON  []byte
			networkInfoJSON []byte
		)

		if err := rows.Scan(&hostInfoJSON, &cpuInfoJSON, &memoryInfoJSON, &networkInfoJSON); err != nil {
			return nil, fmt.Errorf("扫描数据失败: %v", err)
		}

		// 解析各个字段
		if err := json.Unmarshal(hostInfoJSON, &info.HostInfo); err != nil {
			return nil, fmt.Errorf("解析主机信息失败: %v", err)
		}
		if err := json.Unmarshal(cpuInfoJSON, &info.CpuInfo); err != nil {
			return nil, fmt.Errorf("解析CPU信息失败: %v", err)
		}
		if err := json.Unmarshal(memoryInfoJSON, &info.MemoryInfo); err != nil {
			return nil, fmt.Errorf("解析内存信息失败: %v", err)
		}
		if err := json.Unmarshal(networkInfoJSON, &info.NetworkInfo); err != nil {
			return nil, fmt.Errorf("解析网络信息失败: %v", err)
		}
	}

	return &info, nil
}

// GetLastSystemInfo 获取最新的系统信息
func (r *TDengineRepository) GetLastSystemInfo(ctx context.Context, hostname string) (*pb.SystemInfo, error) {
	tableName := fmt.Sprintf("%s_system_info", hostname)
	querySQL := fmt.Sprintf(`
		SELECT host_info, cpu_info, memory_info, process_info, network_info
		FROM %s
		ORDER BY created_at DESC
		LIMIT 1
	`, tableName)

	var (
		hostInfoJSON    []byte
		cpuInfoJSON     []byte
		memoryInfoJSON  []byte
		processInfoJSON []byte
		networkInfoJSON []byte
	)

	err := r.db.QueryRow(querySQL).Scan(&hostInfoJSON, &cpuInfoJSON, &memoryInfoJSON, &processInfoJSON, &networkInfoJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("未找到系统信息")
		}
		return nil, fmt.Errorf("查询系统信息失败: %v", err)
	}

	var info pb.SystemInfo
	// 解析各个字段
	if err := json.Unmarshal(hostInfoJSON, &info.HostInfo); err != nil {
		return nil, fmt.Errorf("解析主机信息失败: %v", err)
	}
	if err := json.Unmarshal(cpuInfoJSON, &info.CpuInfo); err != nil {
		return nil, fmt.Errorf("解析CPU信息失败: %v", err)
	}
	if err := json.Unmarshal(memoryInfoJSON, &info.MemoryInfo); err != nil {
		return nil, fmt.Errorf("解析内存信息失败: %v", err)
	}
	if err := json.Unmarshal(processInfoJSON, &info.ProcessInfo); err != nil {
		return nil, fmt.Errorf("解析进程信息失败: %v", err)
	}
	if err := json.Unmarshal(networkInfoJSON, &info.NetworkInfo); err != nil {
		return nil, fmt.Errorf("解析网络信息失败: %v", err)
	}

	return &info, nil
}

// RequestData 用于接收系统监控数据的请求体
// @Description RequestData 包含所有需要收集的系统信息
type RequestData struct {
	CPUInfo  []model.CPUInfo     `json:"cpu_info"`  // CPU 信息
	HostInfo model.HostInfo      `json:"host_info"` // 主机信息
	MemInfo  model.MemoryInfo    `json:"mem_info"`  // 内存信息
	NetInfo  []model.NetworkInfo `json:"net_info"`  // 网络信息
}

// AddSystemInfo 接收并处理系统监控数据
//
// @Summary 接收系统监控信息（CPU、内存、主机信息等）
// @Description 该API用于接收客户端发送的系统监控数据，并验证token和JWT后将数据存储到数据库中。
// @Tags Monitor
// @Accept json
// @Produce json
// @Param request body RequestData true "请求体包含系统监控数据"
// @Success 201 {object} map[string]string "成功响应"
// @Failure 400 {object} map[string]string "无效的JSON数据或令牌长度错误"
// @Failure 401 {object} map[string]string "授权头缺失或无效的token格式或无效的JWT token"
// @Failure 500 {object} map[string]string "数据库操作失败"
// @Router /monitor [post]

// InsertSystemInfo 插入系统信息到TDengine
func (r *TDengineRepository) InsertSystemInfo(hostname string, hostInfo model.HostInfo, cpuInfo []model.CPUInfo, memoryInfo model.MemoryInfo, networkInfo []model.NetworkInfo) error {
	return model.InsertSystemInfo(hostname, hostInfo, cpuInfo, memoryInfo, networkInfo)
}

// Close 关闭数据库连接
func (r *TDengineRepository) Close() error {
	return r.db.Close()
}

// CleanHostData 清理主机相关的所有数据
func (r *TDengineRepository) CleanHostData(ctx context.Context, hostname string) error {
	// 删除主机相关的所有表
	tables := []string{
		fmt.Sprintf("%s_system_info", hostname),
		fmt.Sprintf("%s_cpu_info", hostname),
		fmt.Sprintf("%s_memory_info", hostname),
		fmt.Sprintf("%s_process_info", hostname),
		fmt.Sprintf("%s_network_info", hostname),
	}

	for _, table := range tables {
		query := fmt.Sprintf("DROP TABLE IF EXISTS %s", table)
		if _, err := r.db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("删除表 %s 失败: %v", table, err)
		}
	}

	return nil
}
