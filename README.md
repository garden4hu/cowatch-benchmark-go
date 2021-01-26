# CoWatchBenchmark-go
CoWatchBenchmark-go is a tool written by [Go](http://golang.org/) for benchmarking the performance of the cowatch server. 

## Arguments

CoWatchBenchmark-go supports the following commands:

```txt
  -c string
        [Mandatory] configure: configure file
  -cr string
        [Mandatory] remote configure: remote configure. No coexistence with -c
  -f int
        frequency: frequency of sending message (default 1000)
  -h string
        [Mandatory] host: address of coWatch server. schema://host (default "http://localhost:80")
  -l int
        message length: size of a message (default 1024)
  -r int
        room size: number of room to create (default 1024)
  -rs string
        [Mandatory] start time for creating room: following RFC3339. For example: 2017-12-08T00:08:00.00+08:00
  -th int
        http timeout(1~60s): http request timeout for create room (default 25)
  -tw int
        websocket timeout(1~60s): websocket request timeout for create user (default 45)
  -u int
        user size: maximum number of user in room (default 1000)
  -us string
        [Mandatory] start time for creating user: following RFC3339. For example: 2017-12-08T00:08:00.00+08:00
  -v int
        verbose log enable:1, disable(default):0 

```

## Configure file mode

The configure file is json style. The full content of the configure file as follow:

```json
    "host": "http://server_host:80",
    "room": 2,
    "user": 10,
    "msg_len": 1024,
    "msg_frequency": 10,
    "log_enable": 0,
    "http_timeout": 25,
    "websocket_timeout": 45,
    "start_time_room": "2021-01-20T21:34:00.00+08:00",
    "start_time_user": "2021-01-20T21:35:00.00+08:00"
```

**Configure file mode will override other arguments**

### Local configure file
CoWatchBenchmark-go can start with a configure file which locates in local driver. For example:
```bash
./CoWatchBenchmark -c ./config.json # config.json and CoWatchBenchmark are in the same directory.
# or
./CoWatchBenchmark -c /etc/cowatch/config.json
```

### Remote configure file
CoWatchBenchmark-go can start with a remote configure file. It's very powerful when you want to getRooms multiple jobs in different client mechine. For example:

```bash
./CoWatchBenchmark -cr https://server_host/path/to/your/config
```
Using remote configure file can be start this program parallel in different client by setting the field `start_time_room` and `start_time_user`.

    Note: `start_time_user` should be after `start_time_room`. The time difference shouldn't be less then 60s.

## Cammond line arguements mode
Refer to Arguments block
