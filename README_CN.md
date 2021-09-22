# CoWatchBenchmark-go
CoWatchBenchmark-go 由 [Golang](http://golang.org/) 写出，为了测试 Cowatch Server 的性能。

## Arguments

CoWatchBenchmark-go 支持 命令行参数启动:

```txt
   -c string
        [Mandatory] 本地配置文件
  -cr string
        [Mandatory] 远程配置文件，和本地配置文件不能同时使用
  -v
        [optional] 显示更多输出
```

## 配置文件模式

配置文件为 Json 格式，形如下:

```json
{
  "host": "http://localhost:8080",
  "room": 2,
  "user": 3,
  "msg_len": 1024,
  "msg_frequency": 10,
  "msg_random_send": 0,
  "log_enable": 0,
  "http_timeout": 25,
  "websocket_timeout": 45,
  "start_time_room": "2021-05-13T13:49:00.00+08:00",
  "start_time_user": "2021-05-13T00:10:00.00+08:00",
  "single_client_mode": 1,
  "parallel_mode": 2,
  "app_id": "abcdefghijklmn",
  "ws_request_speed_number": 100,
  "room_expiration_time": 300,
  "sdk_version": "1.0.0-8589-integration-a3d4cd01",
  "createRoomExtraData": {
    "expireTime": 5,
    "hostname": "co-test-golang",
    "maxMember": 4,
    "roomRole": "room_valid_until_expired",
    "hostLeaveRole": "host_transfer_to_second_after_host_leave",
    "ownerBackRole": "host_return_after_owner_back"
  },
  "http_header": {
    "Origin": "google.com",
    "user-agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/93.0.4577.82 Safari/537.36 Edg/93.0.961.52"
  }
}
```

### parameter instruction

| 项 | 类型 | 默认值 | 强制 * | 说明 |
| --- | :---: | :---:  | :---: |--- |
| host | string |  | Y |  服务器地址 schema://address |
| rooms | uint |  1 | N | 要创建的房间数目 |
| users_per_room | uint |  1 | N |  每房间用户数目 |
| message_frequency | uint | 10/m | N |  每分钟发送消息频率 |
| message_length | uint | 48 (bytes) | N |  消息的长度  |
| http_request_timeout  | uint | 25(s) | N |   http 超时时长  |
| websocket_request_timeout | uint | 25(s) | N | websocket 超时时长 | 
| parallel_mode | uint | 0 | N | 请求服务器的方式 <br>0 顺序请求 <br>1 并发请求  <br>2 批量请求 |
| start_time_for_create_rooms | string |   | O |  只用于并发请求, 举例： "2021-05-13T13:49:00.00+08:00" |
| start_time_for_create_users | string |   | O |  只用于并发请求, 举例： "2021-05-13T13:50:00.00+08:00" |
| single_client_mode | uint | 1 | N | 0: 多点测试 <br>1: 单点测试 |
| app_id | string |  | Y | rtc token  |
| ws_request_speed_number_for_mode_2 | int | | O | 每批次（并发）请求的数目 |
| room_expiration_time_in_second | uint | 300s | Y | WS 在线时长 |
| sdk_version | string |  | Y | the sdk version |
| createRoomExtraData | json | | Y | 创建房间时 http post 的消息体可以增加的自定义参数。<br>注意，可以不添加任何参数，tag 必须存在 |
| http_header | json | | Y  | 自定义的 http header <br>注意，可以不添加任何参数，tag 必须存在 |

* O 表示可选，但是其依赖于其他参数。
注意：参数列表中的所有参数均需要被放入配置文件。下面的举例中使用 `CoWatchBenchmark` 作为可执行程序的名称。

### 本地配置文件
本地加载配置文件启动
```bash
./CoWatchBenchmark -c ./config.json # config.json and CoWatchBenchmark are in the same directory.
# or
./CoWatchBenchmark -c /etc/cowatch/config.json
```

### Remote configure file
远程加载配置文件启动。该模式主要用于多点定时启动对 cowatch server 的 http/ws 请求。

```bash
./CoWatchBenchmark -cr https://server_host/path/to/your/config
```

### 请求模式

#### 请求顺序：

从请求顺序的角度而言：

* 顺序请求 ：所有 http/WS 请求都是顺序的，只用于容量测试
* 并发请求 ：所有 http/WS 并发请求，用于压力测试
* 批次请求 ：顺序请求的改进。缓解顺序请求效率过低的请求。但当每批次请求数目过大时，就会成为并发请求

#### 规模：

从请求规模而言：

* 单点测试 ： 单个本程序运行
* 多点测试 ： 多个本程序于多台客户机运行

### 使用举例

#### 单点测试
```json
"single_client_mode": 1
```

#### 多点测试

```json
"single_client_mode": 0
```


#### 顺序请求

```json
"parallel_mode":0
```

#### 并发请求

并发请求必须设置时间，且开始创建用户时刻需明显晚于开始创建房间的时刻（时间间隔是一个经验值，必要条件是保证在开始创建用户的时候，所有房间的创建过程都已经结束）。

```json
"start_time_for_create_rooms": "2021-05-13T13:49:00.00+08:00",
"start_time_for_create_users": "2021-05-13T13:50:00.00+08:00",
"parallel_mode" : 1
```


#### 批次创建请求
批次创建请求用户负载容量测试，因为顺序创建请求可能速率太低了，所以可以在保证请求成功的前提下，批量并发请求。这样可以在保证请求创建成功的前提下提高创建的速度。
下面的例子是没批次 100 个用户发起连接。注意：这100个用户是并发请求的。

```json
"parallel_mode" : 2,
"ws_request_speed_number_for_mode_2": 100
```


## 创建房间的 HTTP 请求结构体中，添加自定义字段
在配置文件中有一个 `createRoomExtraData` 字段，这个字段可以为 POST 的 JSON 结构体中添加自定义的字段。 **注意，字段仅支持 string 和数字，数组和对象会被忽略掉**

### FAQ

1. 参数中关于时间的参数主要用于多点启动，单点程序不用设置关于开始时间的参数。
2. 终端中输出的用户在线数目是实时的，当用户掉线的时候，log 会输出相关的信息。
3. ws_request_speed_number_for_mode_2 参数需要 parallel=2 来配合
