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
  "host": "http://server_host:80",
  "room": 2,
  "user": 10,
  "msg_len": 1024,
  "msg_frequency": 10,
  "log_enable": 0,
  "http_timeout": 25,
  "websocket_timeout": 45,
  "start_time_room": "2021-01-20T21:34:00.00+08:00",
  "start_time_user": "2021-01-20T21:35:00.00+08:00",
  "single_client_mode": 1,
  "parallel_mode": 2,
  "app_id": "8dad41adda7a4d939aa1aae8484c3981",
  "ws_request_speed_number": 100,
  "ws_online_duration_in_second": 1200
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
| single_client_mode | uint | 1 | N | 只用于单点测试 |
| app_id | string |  | Y | rtc token  |
| ws_request_speed_number_for_mode_2 | int | | O | 每批次（并发）请求的数目 |
| room_expiration_time_in_second | uint | 300s | Y | WS 在线时长 |
| sdk_version | string |  | Y | the sdk version |
| createRoomExtraData | json | | Y | 创建房间时 http post 的消息体可以增加的自定义参数。<br>注意，可以不添加任何参数，tag 必须存在 |
| http_header | json | | Y  | 自定义的 http header <br>注意，可以不添加任何参数，tag 必须存在 |

* O 表示可选，但是其依赖于其他参数。
注意：配置文件具有较高的优先级（ 即不要 CLI 和 配置文件混合使用 ）。下面的举例中使用 `CoWatchBenchmark` 作为可执行程序的名称。

### 本地配置文件
本地加载配置文件启动
```bash
./CoWatchBenchmark -c ./config.json # config.json and CoWatchBenchmark are in the same directory.
# or
./CoWatchBenchmark -c /etc/cowatch/config.json
```

### Remote configure file
远程加载配置文件启动。该模式用于多点定时启动对 cowatch server 的 http/ws 请求。

```bash
./CoWatchBenchmark -cr https://server_host/path/to/your/config
```

### 使用举例

#### 单点启动并发请求
```bash
./CoWatchBenchmark -host "https://server_host:port" -rtcID "abcdefgh" -standalone 1 -parallel 1 
```

#### 多点启动请求

多点并发请求必须设置时间，且开始创建用户时刻需明显晚于开始创建房间的时刻（时间间隔是一个经验值，必要条件是保证在开始创建用户的时候，所有房间的创建过程都已经结束）。
```bash
./CoWatchBenchmark -host "https://server_host:port" -rtcID "abcdefgh" -standalone 1 -parallel 1 -parallelStartTimeRoom "2021-01-20T21:34:00.00+08:00" -parallelStartTimeUser "2021-01-20T21:35:00.00+08:00"
```

#### 批次创建请求
批次创建请求用户负载容量测试，因为顺序创建请求可能速率太低了，所以可以在保证请求成功的前提下，批量并发请求。这样可以在保证请求创建成功的前提下提高创建的速度。
下面的例子就是批量(50个房间的用户并发创建 Websocket 请求) 连续加入房间。
```bash
./CoWatchBenchmark -host "https://server_host:port" -rtcID "abcdefgh" -standalone 1 -parallel 2 -wsCon 50
```

## 创建房间的 HTTP 请求结构体中，添加自定义字段
在配置文件中有一个 `createRoomExtraData` 字段，这个字段可以为 POST 的 JSON 结构体中添加自定义的字段。 **注意，字段仅支持 string 和数字，数组和对象会被忽略掉**

### FAQ

1. 参数中关于时间的参数主要用于多点启动，单点程序不用设置关于开始时间的参数。
2. 终端中输出的用户在线数目是实时的，当用户掉线的时候，log 会输出相关的信息。
3. ws_request_speed_number_for_mode_2 参数需要 parallel=2 来配合
