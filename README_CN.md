# CoWatchBenchmark-go
CoWatchBenchmark-go 由 [Golang](http://golang.org/) 写出，为了测试 Cowatch Server 的性能。

## Arguments

CoWatchBenchmark-go 支持 命令行参数启动:

```txt
  -c string
        [强制] configure: 本地配置文件
  -cr string
        [强制] remote configure: 远程配置文件，应为一个 Internet 资源
  -host string
        [强制] host: address of coWatch server. schema://host
  -httpTimeout int
        http timeout(1~60s): http request timeout for create room (default 25)
  -msgFreq int
        frequency: frequency of sending message per minute (default 10)
  -msglen int
        message length: size of a message (default 48)
  -parallel int
        [强制] mode for socket requesting server.1 代表并发请求, 0 代表串行请求, 2 代表 批量请求（既每次并发放请求一定数目，该过程则是顺序的） (default 1)
  -parallelStartTimeRoom string
        [强制] 开始创建房间的时间 (值为空或不设置代表立刻启动): RFC3339 格式. For example: 2017-12-08T00:08:00.00+08:00
  -parallelStartTimeUser string
        [强制] 开始创建用户的时间 (值为空或不设置代表立刻启动): RFC3339 格式. For example: 2017-12-08T00:08:00.00+08:00
  -room int
        room size: 要创建的房间数量 (default 10)
  -rtcID string
        [强制] webrtc app id
  -standalone int
        [强制] 为 1 则表示单点运行该程序。0 则表示在多点运行程序（可以理解为分布式） (default 1)
  -user int
        user size: 每个房间的用户数目 (default 10)
  -v int
        verbose log enable:1, disable(default):0
  -websocketTimeout int
        websocket timeout(1~60s): websocket request timeout for create user (default 45)
  -wsCon int
        仅用于 parallel = 2 的情况下，表示每批次有多少房间的用户请求加入房间 (default 1)
  -wsOnlineDuration int
        当所有用户都加入房间后，用户通信时间，到时退出。单位为秒(s) (default 300)
```

## 配置文件模式

配置文件为 Json 格式，一个例子:

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
在配置文件中有一个 `createRoomExtraField` 字段，这个字段可以为 POST 的 JSON 结构体中添加自定义的字段。 **注意，字段仅支持 string 和数字，数组和对象会被忽略掉**

### FAQ

1. 参数中关于时间的参数主要用于多点启动，单点程序不用设置关于开始时间的参数。
2. 终端中输出的用户在线数目是实时的，当用户掉线的时候，log 会输出相关的信息。
3. wsCon 参数需要 parallel=2 来配合
