# CoWatchBenchmark-go
CoWatchBenchmark-go is a tool written by [Go](http://golang.org/) for benchmarking the performance of the cowatch server. 

## Arguments

CoWatchBenchmark-go supports the following commands:

```bash
-c string
        configure: configure file
-cr string
        remote configure: remote configure. http(s)
-freq int
        frequency of sending message (default 1000)
-host string
        host: address of coWatch server. schema://host
-len int
        message length: size of a message (default 1024)
-log int
        log enable:1, disable:0
-room int
        room size: number of room to create (default 1024)
-sr string
        start time for creating room, following RFC3339. For example: 2017-12-08T00:08:00.00+08:00 . If not set, start instant.
-su string
        start time for creating user, following RFC3339. For example: 2017-12-08T00:08:00.00+08:00 . If not set, start instant.
-t int
        create room timeout: http request timeout (default 25)
-user int
        user size: maximum number of user in room (default 1000)

```

## Configure file mode

The format of the configure file is json. The full content of the configure file as follow:

```json
{
    "host": "schema://server_host",
    "room": 2,
    "user": 10,
    "msg_len": 1024,
    "msg_frequency": 10,
    "log_enable": 0,
    "timeout": 25,
    "start_time_room": "2021-01-20T21:34:00.00+08:00",
    "start_time_user": ""
}
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
CoWatchBenchmark-go can start with a remote configure file. It's very powerful when you want to run multiple jobs in different client mechine. For example:

```bash
./CoWatchBenchmark -cr https://server_host/path/to/your/config
```
Using remote configure file can be start this program parallel in different client by setting the field `start_time_room` and `start_time_user`.

    Note: `start_time_user` should be after `start_time_room`. The duration shouldn't be less then 30s.

## Cammond line arguements mode
Refer to Arguments block
