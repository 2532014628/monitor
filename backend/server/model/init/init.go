package init

import (
	"bufio"
	u "cmd/server/model/user"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	//"regexp"
	"strings"

	"github.com/go-redis/redis/v8"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// 属性均用驼峰命名转换后的含_的，表名就不含_。
const createTableSQL = `
-- roles 表
CREATE TABLE IF NOT EXISTS roles (
    id SERIAL PRIMARY KEY,
    role_name VARCHAR(50) UNIQUE NOT NULL,
    description TEXT
);

-- users 表
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name VARCHAR UNIQUE NOT NULL,
    email VARCHAR UNIQUE NOT NULL,
    password VARCHAR NOT NULL,
    is_verified BOOLEAN DEFAULT FALSE,
    role_id INT REFERENCES roles(id) DEFAULT 0,
    company_id INT DEFAULT 0,
	token TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- company 表
CREATE TABLE IF NOT EXISTS companies (
    id SERIAL PRIMARY KEY,
    name VARCHAR UNIQUE NOT NULL,
	memberNum int DEFAULT 0,
	systemNum int DEFAULT 0,
    admin_id INT REFERENCES users(id) DEFAULT 0,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
	
-- host表
CREATE TABLE IF NOT EXISTS host_info (
	id SERIAL PRIMARY KEY,
    user_name VARCHAR, -- REFERENCES users(name),
	host_name VARCHAR(255)  UNIQUE,
	company_id INT, -- REFERENCES company(id),
	os TEXT NOT NULL,
	platform TEXT NOT NULL,
	kernel_arch TEXT NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP -- TIMESTAMP WITH TIME ZONE 加上时区
);

-- system_info表
CREATE TABLE IF NOT EXISTS system_info (
	id SERIAL PRIMARY KEY,
	host_info_id INT, -- REFERENCES host_info(id),
	host_name VARCHAR(255), -- REFERENCES host_info(host_name),
	cpu_info JSONB,
	memory_info JSONB,
	process_info JSONB,
	network_info JSONB,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- token表
CREATE TABLE IF NOT EXISTS hostandtoken (
	id SERIAL PRIMARY KEY,
	host_name VARCHAR(255) , -- REFERENCES host_info(host_name),
	token TEXT NOT NULL,
	last_heartbeat TIMESTAMP DEFAULT NOW(),
	status VARCHAR(10) DEFAULT 'offline'
);

-- 在system_info表的host_info_id字段上创建索引，加速通过主机ID查找系统信息
-- CREATE INDEX IF NOT EXISTS idx_system_info_host_info_id ON system_info(host_info_id);

-- 对于system_info表中的JSONB字段(cpu_info, memory_info等)，如果需要根据某些键值进行查询，
-- 可以考虑创建GIN (Generalized Inverted Index) 索引，例如：
-- 假设经常需要基于cpu_info内的某个键（如percent）来查询
CREATE INDEX IF NOT EXISTS idx_system_info_cpu_percent ON system_info USING GIN ((cpu_info->'percent') jsonb_path_ops);

-- 在hostandtoken表的host_name字段上创建索引，加速主机名查找
CREATE INDEX IF NOT EXISTS idx_hostandtoken_host_name ON hostandtoken(host_name);

-- 如果经常按last_heartbeat查询或排序，可以在此字段上创建索引
CREATE INDEX IF NOT EXISTS idx_hostandtoken_last_heartbeat ON hostandtoken(last_heartbeat);
`

// cpu_info示例，每次一新的数据就追加进json里面，这样可以保存多个时间戳的数据
// [
//   {
//     "data": [
//       {
//         "id": 0,
//         "percent": 25.5,
//         "cores_num": 6,
//         "model_name": "Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz"
//       },
//       {
//         "id": 0,
//         "percent": 25.5,
//         "cores_num": 6,
//         "model_name": "Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz"
//       }
//     ],
//     "time": "2025-03-11T13:13:30Z"
//   },
//   ……
// ]

var DB *gorm.DB

var CTX = context.Background()
var RDB *redis.Client

func InitRedis() error {
	redisAddr := os.Getenv("REDIS_ADDR")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisDBstr := os.Getenv("REDIS_DB")
	redisDB, err := strconv.Atoi(redisDBstr)
	if err != nil {
		log.Fatalf("Failed to parse Redis DB number: %v", err)
		return err
	}

	if redisAddr == "" {
		log.Fatal("Redis configuration is missing")
		return fmt.Errorf("Redis configuration is missing")
	}

	RDB = redis.NewClient(&redis.Options{
		Addr:     redisAddr,     // Redis地址
		Password: redisPassword, // 无密码
		DB:       redisDB,       // 使用默认DB
	})

	// 测试连接
	_, err = RDB.Ping(CTX).Result()
	if err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
		return err
	}
	return nil
}

// ConnectDatabase 连接到数据库
func ConnectDatabase() error {
	var err error

	// 获取数据库连接信息
	dsn := fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"))

	// 使用gorm打开数据库连接
	DB, err = gorm.Open(postgres.Open(dsn))
	if err != nil {
		return err // 返回连接错误
	}
	return nil
}

// InitDB 初始化数据库，创建所需的表
func InitDB() error {
	if DB == nil {
		return fmt.Errorf("database connection is not initialized") // 检查数据库连接是否已初始化
	}

	tx := DB.Begin() // 开始事务
	if tx.Error != nil {
		return tx.Error // 返回事务错误
	}

	if err := tx.Exec(createTableSQL).Error; err != nil {
		tx.Rollback() // 回滚事务
		return err    // 返回创建表时的错误
	}

	if err := tx.Commit().Error; err != nil {
		return err // 返回提交事务时的错误
	}

	return nil
}

func isValidJSON(data string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(data), &js) == nil
}

// InitDBData 初始化数据库的基本数据
func InitDBData() error {
	if DB == nil {
		return fmt.Errorf("database connection is not initialized") // 检查数据库连接是否已初始化
	}

	tx := DB.Begin() // 开始事务
	if tx.Error != nil {
		return tx.Error // 返回事务错误
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback() // 如果发生panic，回滚事务
		}
	}()

	var user u.User
	result := tx.Where("name=?", "root").First(&user) // 查找用户名为root的用户

	if result.Error == nil {
		log.Printf("Root already exists") // 用户已存在
		tx.Commit()                       // 提交事务
		return nil
	} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		log.Printf("Failed to find user: %v", result.Error)
		tx.Rollback()       // 回滚事务
		return result.Error // 返回查找用户错误
	}

	// 插入角色数据
	if err := insertRoles(tx); err != nil {
		tx.Rollback() // 回滚事务
		return err    // 返回插入角色时的错误
	}
	fmt.Println("1---------------")

	// 插入用户数据
	if err := insertUsers(tx); err != nil {
		tx.Rollback() // 回滚事务
		return err    // 返回插入用户时的错误
	}
	fmt.Println("2---------------")

	// 插入公司数据
	if err := insertCompanies(tx); err != nil {
		tx.Rollback() // 回滚事务
		return err    // 返回插入用户时的错误
	}
	fmt.Println("3---------------")

	// 插入 host_info 数据
	if err := insertHostInfo(tx); err != nil {
		tx.Rollback() // 回滚事务
		return err    // 返回插入主机信息时的错误
	}
	fmt.Println("4---------------")

	// 插入 system_info 数据
	if err := insertSystemInfo(tx); err != nil {
		tx.Rollback() // 回滚事务
		return err    // 返回插入系统信息时的错误
	}
	fmt.Println("5---------------")

	// 插入 hostandtoken 数据
	if err := insertHostAndToken(tx); err != nil {
		tx.Rollback() // 回滚事务
		return err    // 返回插入 token 信息时的错误
	}
	fmt.Println("6---------------")

	if err := tx.Commit().Error; err != nil {
		return err // 返回提交事务时的错误
	}

	return nil
}

// insertRoles 函数从 roles.txt 文件中读取角色数据
func insertRoles(tx *gorm.DB) error {
	file, err := os.Open("asset/example/roles.txt")
	if err != nil {
		return fmt.Errorf("failed to open roles file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			return fmt.Errorf("invalid line format: %s", line)
		}

		id := parts[0]
		roleName := parts[1]
		description := parts[2]

		if err := tx.Exec("INSERT INTO roles (id, role_name, description) VALUES (?, ?, ?)", id, roleName, description).Error; err != nil {
			return fmt.Errorf("failed to insert role %s: %w", roleName, err)
		}
	}
	return scanner.Err() // 返回扫描器的错误（如果有）
}

// insertUsers 函数从 users.txt 文件中读取用户数据
func insertUsers(tx *gorm.DB) error {
	file, err := os.Open("asset/example/users.txt")
	if err != nil {
		return fmt.Errorf("failed to open users file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// 检查是否以 "//" 开头
		if strings.HasPrefix(line, "//") {
			fmt.Println("Encountered a comment line, exiting the loop.")
			break // 退出循环
		}
		parts := strings.Split(line, ",")
		if len(parts) < 5 {
			return fmt.Errorf("invalid line format: %s", line)
		}
		name := parts[0]
		email := parts[1]
		password := parts[2]
		roleID := parts[3]
		companyID := parts[4]

		if err := tx.Exec("INSERT INTO users (name, email, password, role_id, company_id) VALUES (?, ?, ?, ?, ?)", name, email, password, roleID, companyID).Error; err != nil {
			return fmt.Errorf("failed to insert user %s: %w", name, err)
		}
	}
	return scanner.Err()
}

// insertHostInfo 函数从 host_info.txt 文件中读取主机信息
func insertHostInfo(tx *gorm.DB) error {
	file, err := os.Open("asset/example/host_info.txt")
	if err != nil {
		return fmt.Errorf("failed to open host_info file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// 检查是否以 "//" 开头
		if strings.HasPrefix(line, "//") {
			fmt.Println("Encountered a comment line, exiting the loop.")
			break // 退出循环
		}
		parts := strings.Split(line, ",")
		if len(parts) < 6 {
			return fmt.Errorf("invalid line format: %s", line)
		}

		userName := parts[0]
		hostname := parts[1]
		companyId := parts[2]
		os := parts[3]
		platform := parts[4]
		kernelArch := parts[5]

		if err := tx.Exec("INSERT INTO host_info (user_name, host_name, company_id, os, platform, kernel_arch) VALUES (?, ?, ?, ?, ?, ?)", userName, hostname, companyId, os, platform, kernelArch).Error; err != nil {
			return fmt.Errorf("failed to insert host_info for %s: %w", hostname, err)
		}
	}
	return scanner.Err()
}

// insertUsers 函数从 users.txt 文件中读取用户数据
func insertCompanies(tx *gorm.DB) error {
	file, err := os.Open("asset/example/companies.txt")
	if err != nil {
		return fmt.Errorf("failed to open companies file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// 检查是否以 "//" 开头
		if strings.HasPrefix(line, "//") {
			fmt.Println("Encountered a comment line, exiting the loop.")
			break // 退出循环
		}
		parts := strings.Split(line, ",")
		if len(parts) < 5 {
			return fmt.Errorf("invalid line format: %s", line)
		}

		name := parts[0]
		memberNum := parts[1]
		systemNum := parts[2]
		admin_id := parts[3]
		description := parts[4]
		if err := tx.Exec("INSERT INTO companies (name, memberNum, systemNum, admin_id, description) VALUES (?, ?, ?, ?, ?)", name, memberNum, systemNum, admin_id, description).Error; err != nil {
			return fmt.Errorf("failed to insert user %s: %w", name, err)
		}
	}
	return scanner.Err()
}

// insertSystemInfo 函数从 system_info.txt 文件中读取系统信息
func insertSystemInfo(tx *gorm.DB) error {
	file, err := os.Open("asset/example/system_info.txt")
	if err != nil {
		return fmt.Errorf("failed to open system_info file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// 检查是否以 "//" 开头
		if strings.HasPrefix(line, "//") {
			fmt.Println("Encountered a comment line, exiting the loop.")
			break // 退出循环
		}
		// fmt.Println("Read line:", line) // 输出读取的行（调试用）

		parts := strings.Split(line, ",,")
		// fmt.Println()
		// fmt.Println("len:",len(parts))
		// fmt.Println()
		if len(parts) < 6 {
			return fmt.Errorf("invalid line format: %s", line)
		}
		hostName := parts[0]
		hostInfoID := parts[1]
		cpuInfo := parts[2]
		memoryInfo := parts[3]
		processInfo := parts[4]
		networkInfo := parts[5]
		// fmt.Println("hostName:", hostName)
		// fmt.Println("hostInfoID:", hostInfoID)
		// fmt.Println("cpuInfo:", cpuInfo)
		// fmt.Println("memoryInfo:", memoryInfo)
		// fmt.Println("processInfo:", processInfo)
		// fmt.Println("networkInfo:", networkInfo)

		// 验证每个 JSON 字符串的有效性
		if !isValidJSON(cpuInfo) || !isValidJSON(memoryInfo) || !isValidJSON(processInfo) || !isValidJSON(networkInfo) {
			return fmt.Errorf("invalid JSON data for host %s", hostName)
		}

		// 插入数据库（注意：这里假设数据库表 system_info 的对应字段已经设置为接受 jsonb 类型）
		if err := tx.Exec(
			"INSERT INTO system_info (host_name, host_info_id, cpu_info, memory_info, process_info, network_info) VALUES (?, ?, ?::jsonb, ?::jsonb, ?::jsonb, ?::jsonb)",
			hostName, hostInfoID, cpuInfo, memoryInfo, processInfo, networkInfo,
		).Error; err != nil {
			return fmt.Errorf("failed to insert system info for host %s: %w", hostName, err)
		}
	}
	return scanner.Err() // 返回读取文件的错误（如果有）
}

// insertHostAndToken 函数从 hostandtoken.txt 文件中读取 token 数据
func insertHostAndToken(tx *gorm.DB) error {
	file, err := os.Open("asset/example/hostandtoken.txt")
	if err != nil {
		return fmt.Errorf("failed to open hostandtoken file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// 检查是否以 "//" 开头
		if strings.HasPrefix(line, "//") {
			fmt.Println("Encountered a comment line, exiting the loop.")
			break // 退出循环
		}
		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			return fmt.Errorf("invalid line format: %s", line)
		}

		hostName := parts[0]
		token := parts[1]
		status := parts[2]

		if err := tx.Exec("INSERT INTO hostandtoken (host_name, token, status) VALUES (?, ?, ?)", hostName, token, status).Error; err != nil {
			return fmt.Errorf("failed to insert token for host %s: %w", hostName, err) // 返回详细错误
		}
	}
	return scanner.Err()
}

// -- cpu表
// CREATE TABLE IF NOT EXISTS cpu_info (
// 	id SERIAL PRIMARY KEY,
// 	host_id INT REFERENCES host_info(id),
// 	model_name TEXT NOT NULL,
// 	cores_num INT NOT NULL,
// 	percent NUMERIC(5,2) NOT NULL,
// 	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
// );

// -- memory 表
// CREATE TABLE IF NOT EXISTS memory_info (
// 	id SERIAL PRIMARY KEY,
// 	host_id INT REFERENCES host_info(id),
// 	total NUMERIC(10,2) NOT NULL,
// 	available NUMERIC(10,2) NOT NULL,
// 	used NUMERIC(10,2) NOT NULL,
// 	free NUMERIC(10,2) NOT NULL,
// 	user_percent NUMERIC(5,2) NOT NULL,
// 	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
// 	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
// );

// -- process 表
// CREATE TABLE IF NOT EXISTS process_info (
// 	id SERIAL PRIMARY KEY,
// 	host_id INT REFERENCES host_info(id),
// 	pid INT NOT NULL,
// 	cpu_percent NUMERIC(5,2) NOT NULL,
// 	mem_percent NUMERIC(5,2) NOT NULL,
// 	cmdline TEXT NOT NULL,
// 	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
// );

// -- net_info表
// CREATE TABLE IF NOT EXISTS network_info (
// 	id SERIAL PRIMARY KEY,
// 	host_id INT REFERENCES host_info(id),
// 	bytesrecv BIGINT NOT NULL,
// 	bytessent BIGINT NOT NULL,
// 	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
//);
