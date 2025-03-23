# 用户注册与登录接口说明

## 注册接口

### 请求格式
- **URL**: `/agent/register`
- **Method**: `POST`
- **Content-Type**: `application/json`

### 请求示例
```json 
{
   "name": "yjy01",
   "email": "example@qq.com",
   "password": "12345678" 
 }
 ```
 
### 字段说明
- `name`: 用户名，字符串类型，必填。
- `email`: 用户邮箱，字符串类型，必填。
- `password`: 用户密码，字符串类型，必填。

## 登录接口

### 请求格式
- **URL**: `/agent/login`
- **Method**: `POST`
- **Content-Type**: `application/json`

### 请求示例
```json 
{
  "name": "yjy01",
  "password": "12345678"
}
```
### 字段说明
- `name`: 用户名，字符串类型，必填。
- `password`: 用户密码，字符串类型，必填。

# 列出所有主机信息接口说明

## 接口描述
该接口用于查询当前用户的所有主机信息，支持按时间范围过滤。

## 请求格式
- **URL**: `/agent/list`
- **Method**: `GET`
- **Content-Type**: `application/json`
- **Authorization**: `your_jwt_token`

## 请求参数
| 参数名 | 类型   | 必填 | 说明                                                                 |
|--------|--------|------|--------------------------------------------------------------------|
| from   | string | 否   | 起始时间，格式为 `RFC3339`（如 `2023-01-01T00:00:00Z`），默认为 `1970-01-01T00:00:00Z` |
| to     | string | 否   | 结束时间，格式为 `RFC3339`（如 `2023-12-31T23:59:59Z`），默认为 `9999-12-31T23:59:59Z` |

## 响应格式
- **Content-Type**: `application/json`
- **响应示例**:
```json
[ 
  {
    "id": 2, 
    "host_name": "my-host", 
    "os": "Linux", 
    "platform": "Ubuntu 20.04", 
    "kernel_arch": "x86_64", 
    "host_info_created_at": "2025-03-10T18:17:16.189247Z", 
    "token": ""
  }, 
  { 
    "id": 3, 
    "host_name": "web-server", 
    "os": "Linux", 
    "platform": "CentOS 7", 
    "kernel_arch": "x86_64", 
    "host_info_created_at": "2024-05-15T12:34:56.789012Z", 
    "token": ""
  }, 
  {
    "id": 4, 
    "host_name": "db-server", 
    "os": "Linux", 
    "platform": "Debian 10", 
    "kernel_arch": "x86_64", 
    "host_info_created_at": "2024-08-20T09:45:23.456789Z", 
    "token": ""
  }, 
  { 
    "id": 5, 
    "host_name": "dev-machine", 
    "os": "Windows", 
    "platform": "Windows 10", 
    "kernel_arch": "x86_64", 
    "host_info_created_at": "2024-11-25T15:30:10.123456Z", 
    "token": ""
  }, 
  { 
    "id": 6, 
    "host_name": "test-server", 
    "os": "Linux", 
    "platform": "Fedora 33", 
    "kernel_arch": "x86_64", 
    "host_info_created_at": "2025-01-30T20:10:05.987654Z", 
    "token": ""
  } 
]
```
### 字段说明
| 字段名               | 类型   | 说明                     |
|----------------------|--------|------------------------|
| id                   | int    | 主机唯一标识             |
| host_name             | string | 主机名                   |
| os                   | string | 操作系统                 |
| platform             | string | 操作系统版本             |
| kernel_arch          | string | 内核架构                 |
| host_info_created_at | string | 主机信息创建时间         |
| token                | string | 主机令牌（暂未使用）     |

## 注意事项
1. 请确保在请求头中正确设置 `Content-Type` 为 `application/json`。
2. 时间参数 `from` 和 `to` 必须符合 `RFC3339` 格式。
3. 如果未提供时间参数，默认查询范围为 `1970-01-01T00:00:00Z` 到 `9999-12-31T23:59:59Z`。

# 查询单个机器具体信息接口说明

## 接口描述
该接口用于查询特定主机的详细信息，包括 CPU、内存、网络和进程等数据。

## 请求格式
- **URL**: `/monitor/:hostname`(hostname填写要查询的主机名)
- **Method**: `GET`
- **Content-Type**: `application/json`

## 请求参数
### URL 参数
| 参数名     | 类型   | 必填 | 说明         |
|------------|--------|------|--------------|
| hostname  | string | 是   | 主机名       |

### Query 参数
| 参数名 | 类型   | 必填 | 说明                                                                 |
|--------|--------|------|--------------------------------------------------------------------|
| type   | string | 否   | 查询类型，默认为 `all`（返回所有信息），可选值：`cpu`, `memory`, `net`, `process` |
| from   | string | 否   | 起始时间，格式为 `RFC3339`（如 `2023-01-01T00:00:00Z`），默认为 `1970-01-01T00:00:00Z` |
| to     | string | 否   | 结束时间，格式为 `RFC3339`（如 `2023-12-31T23:59:59Z`），默认为 `9999-12-31T23:59:59Z` |

## 响应格式
- **Content-Type**: `application/json`
  - **响应示例**:
```json 
{
  "cpu": 
  [
    {
      "data": 
            { 
              "cores_num": 6, 
              "cpu_info_created_at": "0001-01-01T00:00:00Z", 
              "id": 0, 
              "model_name": "Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz", 
              "percent": 25.5
            }, 
      "time": "2025-03-10T10:17:16Z"
    } 
  ], 
  "host": 
  {
    "host_info_created_at": "2025-03-10T18:17:16.189247Z", 
    "host_name": "my-host", 
    "id": 2, 
    "kernel_arch": "x86_64", 
    "os": "Linux", 
    "platform": "Ubuntu 20.04"
  }, 
  "memory": 
  [ 
    { 
      "data": { 
        "available": "8GB",
        "free": "8GB",
        "id": 0,
        "mem_info_created_at": "0001-01-01T00:00:00Z",
        "total": "16GB",
        "used": "8GB",
        "user_percent": 50
      },
      "time": "2025-03-10T10:17:16Z"
    } 
  ], 
  "net": [ 
    { 
      "data": { 
        "bytes_recv": 1024, 
        "bytes_sent": 2048, 
        "id": 0, 
        "name": "eth0", 
        "net_info_created_at": "0001-01-01T00:00:00Z"
      }, 
      "time": "2025-03-10T10:17:16Z"
    } 
  ], 
  "process": [
    {
      "data": 
      { 
        "cmdline": "/usr/bin/python3", 
        "cpu_percent": 10.5, 
        "id": 0, 
        "mem_percent": 5.5, 
        "pid": 1234, 
        "pro_info_created_at": "0001-01-01T00:00:00Z"
      }, 
      "time": "2025-03-10T10:17:16Z" } 
  ]
}
```
### 字段说明
#### `cpu`
| 字段名               | 类型   | 说明                     |
|----------------------|--------|------------------------|
| cores_num            | int    | CPU 核心数              |
| cpu_info_created_at  | string | CPU 信息创建时间         |
| id                   | int    | CPU 信息唯一标识         |
| model_name           | string | CPU 型号名称             |
| percent              | float  | CPU 使用率               |
| time                 | string | 数据记录时间             |

#### `host`
| 字段名               | 类型   | 说明                     |
|----------------------|--------|------------------------|
| host_info_created_at | string | 主机信息创建时间         |
| host_name             | string | 主机名                   |
| id                   | int    | 主机唯一标识             |
| kernel_arch          | string | 内核架构                 |
| os                   | string | 操作系统                 |
| platform             | string | 操作系统版本             |

#### `memory`
| 字段名               | 类型   | 说明                     |
|----------------------|--------|------------------------|
| available            | string | 可用内存大小             |
| free                 | string | 空闲内存大小             |
| id                   | int    | 内存信息唯一标识         |
| mem_info_created_at  | string | 内存信息创建时间         |
| total                | string | 总内存大小               |
| used                 | string | 已用内存大小             |
| user_percent         | float  | 内存使用率               |
| time                 | string | 数据记录时间             |

#### `net`
| 字段名               | 类型   | 说明                     |
|----------------------|--------|------------------------|
| bytes_recv           | int    | 接收字节数               |
| bytes_sent           | int    | 发送字节数               |
| id                   | int    | 网络信息唯一标识         |
| name                 | string | 网络接口名称             |
| net_info_created_at  | string | 网络信息创建时间         |
| time                 | string | 数据记录时间             |

#### `process`
| 字段名               | 类型   | 说明                     |
|----------------------|--------|------------------------|
| cmdline              | string | 进程命令行               |
| cpu_percent          | float  | 进程 CPU 使用率          |
| id                   | int    | 进程信息唯一标识         |
| mem_percent          | float  | 进程内存使用率           |
| pid                  | int    | 进程 ID                  |
| pro_info_created_at  | string | 进程信息创建时间         |
| time                 | string | 数据记录时间             |

## 注意事项
1. 请确保在请求头中正确设置 `Content-Type` 为 `application/json`。
2. 时间参数 `from` 和 `to` 必须符合 `RFC3339` 格式。
3. 如果未提供时间参数，默认查询范围为 `1970-01-01T00:00:00Z` 到 `9999-12-31T23:59:59Z`。