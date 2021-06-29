package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"
)

var config = flag.String("c", "", "[Mandatory] configure: configure file")
var addr = flag.String("host", "", "[Mandatory] host: address of coWatch server. schema://host")
var room = flag.Int("room", 10, "room size: number of room to create")
var user = flag.Int("user", 10, "user size: maximum number of user in room")
var msgLen = flag.Int("msglen", 48, "message length: size of a message")
var frequency = flag.Int("msgFreq", 10, "frequency: frequency of sending message per minute")
var logSwitch = flag.Int("v", 0, "verbose log enable:1, disable(default):0")
var remoteConfig = flag.String("cr", "", "[Mandatory] remote configure: remote configure. No coexistence with -c")
var httpReqTimeOut = flag.Int("httpTimeout", 25, "http timeout(1~60s): http request timeout for create room")
var wsReqTimeOut = flag.Int("websocketTimeout", 45, "websocket timeout(1~60s): websocket request timeout for create user")
var startTimeCreatingRoom = flag.String("parallelStartTimeRoom", "", "[Mandatory] start time for creating room: following RFC3339. For example: 2017-12-08T00:08:00.00+08:00")
var startTimeCreatingUser = flag.String("parallelStartTimeUser", "", "[Mandatory] start time for creating user: following RFC3339. For example: 2017-12-08T00:08:00.00+08:00")
var parallelMode = flag.Int("parallel", 1, "[Mandatory] mode for socket requesting server.1 means parallel, 0 means serial")
var singleClientMode = flag.Int("standalone", 1, "[Mandatory] set to 1 means run cowatch-benchamrk in one point. 0 means multi-point in the same time")
var appID = flag.String("rtcID", "", "[Mandatory] webrtc app id")
var wsReqSpeed = flag.Int("wsCon", 1, "for parallel mode, it means that the number of room which fire websockets. It should be positive and only valid when parallel_mode=2")
var wsOnlineDuration = flag.Int("wsOnlineDuration", 300, "websocket link survival duration. In second.")

func Init() {
	flag.Parse()
	onlineUser = 0
	analytics = new(statistic)
	webSocketRunningDuration = time.NewTicker(1 * time.Hour)
}

var rm *roomManager
var onlineUser int
var analytics *statistic
var webSocketRunningDuration *time.Ticker

func main() {
	Init()
	// process args
	var configure *Config
	if len(*config) > 0 {
		conf, err := readConfigure(*config)
		if err != nil {
			fmt.Println("failed to read configure, check your path or content of configure file, program will be exited")
			return
		}
		configure = conf
	} else if len(*remoteConfig) > 0 {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr}
		resp, err := client.Get(*remoteConfig)
		if err != nil {
			fmt.Println("[error]  Program failed to download configure file from remote source, will be exited")
			return
		}
		defer resp.Body.Close()
		b, _ := ioutil.ReadAll(resp.Body)
		var conf Config
		if err := json.Unmarshal(b, &conf); err != nil {
			fmt.Println("failed to read configure, check your path or content of configure file, program will be exited")
			return
		}
		configure = &conf
	} else {
		configure = &Config{Host: *addr, Room: *room, User: *user, Len: *msgLen, Freq: *frequency, Log: *logSwitch, HttpTimeOut: *httpReqTimeOut, WSTimeOut: *wsReqTimeOut, StartTimeRoom: *startTimeCreatingRoom, StartTimeUser: *startTimeCreatingUser, ParallelMode: *parallelMode, SingleClientMode: *singleClientMode, AppID: *appID, WsReqConcurrency: *wsReqSpeed, OnlineTime: *wsOnlineDuration}
	}

	_, _ = fmt.Fprintf(os.Stdout, "\ninput info:\n server address:\t %s\n number of room:\t %d\n users per room:\t %d\n message length:\t %d\n message frequency:\t %d\n log enable:\t %d\n single client mode: \t %d\n parallel_mode:\t %d\n ws_request_speed_number:\t %d\n ws_online_duration_in_second:\t%d\n\n", configure.Host, configure.Room, configure.User, configure.Len, configure.Freq, configure.Log, configure.SingleClientMode, configure.ParallelMode, configure.WsReqConcurrency, configure.OnlineTime)
	if configureCheck(configure) != nil {
		fmt.Println("\n main program exit now.")
		return
	}
	if configure.Log == 0 {
		log.SetFlags(0)
		log.SetOutput(ioutil.Discard)
	}
	rm = newRoomManager(configure.Host, configure.Room, configure.User, configure.Len, configure.Freq, configure.HttpTimeOut, configure.WSTimeOut, configure.AppID, configure.SingleClientMode, configure.ParallelMode)
	defer rm.Close()
	// register system interrupt
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	defer fmt.Println("main exit")
	defer time.Sleep(3 * time.Second)
	defer cancel()
	if configure.ParallelMode == 1 {
		go getRoomsParallel(configure, ctx)
	} else {
		go NonInstanceLoading(configure, ctx)
	}
	// ticker used for get statistics information
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		// create room
		select {
		case <-interrupt:
			log.Println("interrupt by user")
			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			return
		case <-ticker.C:
			go printLogMessage(rm)
			ticker.Reset(1 * time.Second)
			break
		case t := <-rm.notifyUsersAdd:
			onlineUser += t
			break
		case <-webSocketRunningDuration.C:
			webSocketRunningDuration.Stop()
			fmt.Println("The program exits at time")
			return
		}
	}
}

func readConfigure(path string) (*Config, error) {
	path = filepath.Clean(path)
	if filepath.IsAbs(path) == false {
		dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
		path = filepath.Join(dir, path)
	}

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var conf Config
	if err := json.Unmarshal(b, &conf); err != nil {
		return nil, err
	}
	return &conf, nil
}

func configureCheck(conf *Config) error {

	checkTime := func(tim string) error {
		_, err := time.Parse(time.RFC3339, tim)
		if err != nil {
			fmt.Println("[ERROR] time string is invalid. A valid example is looks like: 2017-12-08T00:08:00.00+08:00")
			return nil
		}
		return nil
	}
	err := errors.New("\n configure check failed \n")
	if conf.Host == "" {
		fmt.Println("[ERROR] Host invalid")
		return err
	} else if conf.Room <= 0 {
		fmt.Println("[ERROR] number of room is invalid")
		return err
	} else if conf.User <= 0 {
		fmt.Println("[ERROR] number of users per room is invalid")
		return err
	} else if conf.Freq <= 0 {
		fmt.Println("[ERROR] number of users per room is invalid, should be positive")
		return err
	} else if conf.SingleClientMode < 0 || conf.SingleClientMode > 1 {
		fmt.Println("[ERROR] single_client_mode is invalid, should be 0 or 1")
		return err
	} else if conf.ParallelMode < 0 || conf.ParallelMode > 2 {
		fmt.Println("[ERROR] parallel_mode is invalid, should be 0, 1, 2")
		return err
	} else if conf.WsReqConcurrency <= 0 {
		fmt.Println("[ERROR] ws_request_speed_number is invalid, should be positive")
		return err
	} else if err = checkTime(conf.StartTimeRoom); err != nil {
		fmt.Println("[ERROR] start_time_room is invalid,", err)
		return err
	} else if err = checkTime(conf.StartTimeUser); err != nil {
		fmt.Println("[ERROR] start_time_user is invalid,", err)
		return err
	} else if conf.ParallelMode == 2 && conf.OnlineTime <= 0 {
		fmt.Println("[ERROR] ws_online_duration_in_second is invalid, should be positive")
		return err
	} else {
		return nil
	}
}

type Config struct {
	Host             string `json:"host"`
	Room             int    `json:"room"`
	User             int    `json:"user"`
	Len              int    `json:"msg_len"`
	Freq             int    `json:"msg_frequency"`
	RandomMsg        int    `json:"msg_random_send"`
	Log              int    `json:"log_enable"`
	AppID            string `json:"app_id"`
	HttpTimeOut      int    `json:"http_timeout"`
	WSTimeOut        int    `json:"websocket_timeout"`
	StartTimeRoom    string `json:"start_time_room"`
	StartTimeUser    string `json:"start_time_user"`
	SingleClientMode int    `json:"single_client_mode"`
	ParallelMode     int    `json:"parallel_mode"`
	WsReqConcurrency int    `json:"ws_request_speed_number"`
	OnlineTime       int    `json:"ws_online_duration_in_second"`
}
