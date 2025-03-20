package config

import (
	"io/ioutil"
	"log"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v2"
)

// DBConfig 用于保存数据库配置
type DBConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Name     string `yaml:"name"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type OSSConfig struct {
	OSS_REGION            string `yaml:"OSS_REGION"`
	OSS_ACCESS_KEY_ID     string `yaml:"OSS_ACCESS_KEY_ID"`
	OSS_ACCESS_KEY_SECRET string `yaml:"OSS_ACCESS_KEY_SECRET"`
	OSS_BUCKET            string `yaml:"OSS_BUCKET"`
}

type RedisConfig struct {
	Addr     string `yaml:"redis_addr"`
	Password string `yaml:"redis_password"`
	DB       string `yaml:"redis_db"`
}

type EMAILConfig struct {
	Name     string `yaml:"email_name"`
	Password string `yaml:"email_password"`
	Url      string `yaml:"base_url"`
}
type SMTPServerConfig struct {
	Host string `yaml:"SMTPServer_host"`
	Port string `yaml:"SMTPServer_port"`
}

// Config 用于保存所有配置项
type Config struct {
	DB         DBConfig         `yaml:"db"`
	OSS        OSSConfig        `yaml:"oss"`
	Redis      RedisConfig      `yaml:"redis"`
	Email      EMAILConfig      `yaml:"email"`
	SMTPServer SMTPServerConfig `yaml:"smtp_server"`
}

// getDBConfigPath 获取数据库配置文件的路径
func getDBConfigPath() string {
	_, filename, _, ok := runtime.Caller(2) // 获取调用者的文件名
	if !ok {
		log.Fatal("无法获取运行时调用者信息")
	}

	// 获取当前文件所在的目录
	currentDir := filepath.Dir(filename)

	// 构建到项目根目录的相对路径
	dbConfigPath := filepath.Join(currentDir, "..", "config", "configs", "config.yaml")

	// 将路径转换为绝对路径并简化路径
	absPath, err := filepath.Abs(dbConfigPath)
	if err != nil {
		log.Fatalf("无法获取绝对路径: %v", err)
	}

	simplifiedPath := filepath.Clean(absPath)

	return simplifiedPath
}

// GetDBConfigPath 返回数据库配置文件的路径
func GetDBConfigPath() string {
	return getDBConfigPath()
}

// LoadConfig 加载配置文件并返回 DBConfig
func LoadConfig() (*Config, error) {
	configPath := GetDBConfigPath()
	yamlFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Printf("读取配置文件失败: %v", err)
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Printf("解析配置文件失败: %v", err)
		return nil, err
	}

	return &config, nil
}
