# CoWatchBenchmark-go
CoWatchBenchmark-go is a tool written by [Go](http://golang.org/) for benchmarking the performance of the cowatch server. 

[中文](./README_CN.md)

## Arguments

CoWatchBenchmark-go supports the following commands:

```txt
  -c string
        [Mandatory] configure: configure file
  -cr string
        [Mandatory] remote configure: remote configure. No coexistence with -c
  -v
        [optional] show verbos output
```

## Configure file mode

The configure file is json style. The full content of the configure file as follows:

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

| entry | type | default value | mandatory * | instruction |
| --- | :---: | :---:  | :---: |--- |
| host | string |  | Y |  address of coWatch server. schema://address |
| rooms | uint |  1 | N | the maximum number of room |
| users_per_room | uint |  1 | N |  users in each room |
| message_frequency | uint | 10/m | N |  frequency of sending message per minute |
| message_length | uint | 48 (bytes) | N |  size of a message  |
| http_request_timeout  | uint | 25(s) | N |  http request timeout for create room  |
| websocket_request_timeout | uint | 25(s) | N | request timeout for websocket | 
| parallel_mode | uint | 0 | N | mode for socket requesting server. <br>0 means serial <br>1 means parallel  <br>2 mean batch |
| start_time_for_create_rooms | string |   | O |  used in parallel = 1 only, such as "2021-05-13T13:49:00.00+08:00" |
| start_time_for_create_users | string |   | O |  used in parallel = 1 only, such as "2021-05-13T13:50:00.00+08:00" |
| single_client_mode | uint | 1 | N |  0: multi-point testing <br>1: single-point testing |
| app_id | string |  | Y | rtc token  |
| ws_request_speed_number_for_mode_2 | int | | O | the number of ws request in a batch |
| room_expiration_time_in_second | uint | 300s | Y | the living time for a room |
| sdk_version | string |  | Y | the sdk version |
| createRoomExtraData | json | | Y | the custom data of body of posting when create the room. <br>Note: the content in it can be empty |
| http_header | json | | Y  | the custom http header. <br>Note: the content in it can be empty |

* O in **mandatory** means options. It depends on other entries.

**Note: Configure file mode will override other arguments**

## Local configure file
CoWatchBenchmark-go can be started with a configure file which locates in local driver. For example:
```bash
./CoWatchBenchmark -c ./config.json # config.json and CoWatchBenchmark are in the same directory.
# or
./CoWatchBenchmark -c /etc/cowatch/config.json
```

## Remote configure file
CoWatchBenchmark-go can also be started with a remote configure file. It's very powerful when you want to getRooms multiple jobs in different client mechine. For example:

```bash
./CoWatchBenchmark -cr https://server_host/path/to/your/config
```
Using remote configure file can be start this program parallel in different client by setting the field `start_time_room` and `start_time_user` when you want to start multi-point jobs.


## Testing mode
### testing scale
From the perspective of test scale, there are two ways of single-point testing and multi-point testing.

* Single point : run one copy of this program
* Multi point : run multi copy of this program in multi host


### request sequence

From the perspective of request sequence, it is divided into sequential requests, concurrent requests and batch requests.

* sequential requests: all http/ws requsets are sequential, i.e. one by one.
* concurrent requests: all http/ws requsets launch in the same time.
* batch requests: it alleviate the inefficiency of sequential requests. Essentially, it changes one request only at a time of sequential requests to several concurrent requests at a time.

Note: If the number of each batch in the batch requests is too large, the batch request will become a concurrent requests.


### samples

#### Single point
```json
"single_client_mode": 1
```

#### Multi point

```json
"single_client_mode": 0
```
#### sequential requests

```json
"parallel_mode":0
```

#### concurrent requests

```json
"start_time_for_create_rooms": "2021-05-13T13:49:00.00+08:00",
"start_time_for_create_users": "2021-05-13T13:50:00.00+08:00",
"parallel_mode" : 1
```

#### batch requests

```json
"parallel_mode" : 2,
"ws_request_speed_number_for_mode_2": 100
```

## Extra parameter of creating room
There is a `createRoomExtraData` field in the config file. It supports adding extra data to the body of HTTP requesting. **You should not add an array or an object into the createRoomExtraData object. That is to say, only string and number are supporting.**

## command line arguments mode
Refer to Arguments block
