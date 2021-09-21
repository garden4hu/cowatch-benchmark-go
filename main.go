package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"
)

var config = flag.String("c", "", "[Mandatory] configure: configure file")
var remoteConfig = flag.String("cr", "", "[Mandatory] remote configure: remote configure. No coexistence with -c")
var verbose = flag.Bool("v", false, "show log")

//var addr = flag.String("host", "", "[Mandatory] host: address of coWatch server. schema://host")
//var room = flag.Int("room", 10, "room size: number of room to create")
//var user = flag.Int("user", 10, "user size: maximum number of user in room")
//var msgLen = flag.Int("msglen", 48, "message length: size of a message")
//var frequency = flag.Int("msgFreq", 10, "frequency: frequency of sending message per minute")
//var logSwitch = flag.Int("v", 0, "verbose log enable:1, disable(default):0")

//var httpReqTimeOut = flag.Int("httpTimeout", 25, "http timeout(1~60s): http request timeout for create room")
//var wsReqTimeOut = flag.Int("websocketTimeout", 45, "websocket timeout(1~60s): websocket request timeout for create user")
//var startTimeCreatingRoom = flag.String("parallelStartTimeRoom", "", "[Mandatory] start time for creating room: following RFC3339. For example: 2017-12-08T00:08:00.00+08:00")
//var startTimeCreatingUser = flag.String("parallelStartTimeUser", "", "[Mandatory] start time for creating user: following RFC3339. For example: 2017-12-08T00:08:00.00+08:00")
//var parallelMode = flag.Int("parallel", 1, "[Mandatory] mode for socket requesting server.1 means parallel, 0 means serial")
//var singleClientMode = flag.Int("standalone", 1, "[Mandatory] set to 1 means run cowatch-benchamrk in one point. 0 means multi-point in the same time")
//var appID = flag.String("rtcID", "", "[Mandatory] webrtc app id")
//var wsReqSpeed = flag.Int("wsCon", 1, "for parallel mode, it means that the number of room which fire websockets. It should be positive and only valid when parallel_mode=2")
//var wsOnlineDuration = flag.Int("wsOnlineDuration", 300, "websocket link survival duration. In second.")

var log = logrus.New()
var logF = logrus.New()

func init() {
	flag.Parse()
	onlineUser = 0
	analytics = new(statistic)
	webSocketRunningDuration = time.NewTicker(1 * time.Hour)
	logF.Formatter = &logrus.JSONFormatter{}
	log.Formatter = &logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	}
	log.SetReportCaller(true)
	log.SetOutput(os.Stderr)
	logF.SetOutput(os.Stdout)
	log.SetLevel(logrus.InfoLevel)
	logF.SetLevel(logrus.InfoLevel)
	if *verbose == true {
		log.SetLevel(logrus.DebugLevel)
		logF.SetLevel(logrus.DebugLevel)
	}
}

var rm *roomManager
var onlineUser int
var analytics *statistic
var webSocketRunningDuration *time.Ticker

func main() {
	// process args
	var configure *Config
	if len(*config) > 0 {
		conf, err := readConfigure(*config)
		if err != nil {
			log.Errorln("failed to read configure, check your path or content of configure file, program will be exited")
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
			log.Errorln("[error]  Program failed to download configure file from remote source, will be exited")
			return
		}
		defer resp.Body.Close()
		b, _ := ioutil.ReadAll(resp.Body)
		var conf Config
		if err := json.Unmarshal(b, &conf); err != nil {
			log.Errorln("failed to read configure file, check your path or make sure the content is ok, program will be exited now")
			return
		}
		configure = &conf
	} else {
		log.Errorln("program doesn't support command, please set the configure file, it will be exited now")
		return
	}
	indent, err := json.MarshalIndent(configure, "", "\t")
	if err == nil {
		log.Infoln("your configure is :\n", string(indent))
	}

	if e := configureCheck(configure); e != nil {
		log.Errorln(e.Error())
		return
	}
	rm = newRoomManager(configure)
	processExtraHttpData(rm, configure)
	defer rm.Close()
	// register system interrupt
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
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
			log.Warnln("interrupt by user")
			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.

			time.Sleep(1 * time.Second)
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
			log.Warnln("The program exits at time")
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

func configureCheck(conf *Config) (e error) {
	checkTime := func(tim string) (err error) {
		_, err = time.Parse(time.RFC3339, tim)
		if err != nil {
			err = errors.New("time string is invalid. A valid example is looks like: 2017-12-08T00:08:00.00+08:00")
		}
		return err
	}
	if conf.Host == "" {
		e = errors.New("[ERROR] Host invalid")
	} else if conf.Room <= 0 {
		e = errors.New("[ERROR] number of room is invalid")
	} else if conf.User <= 0 {
		e = errors.New("[ERROR] number of users per room is invalid")
	} else if conf.Freq <= 0 {
		e = errors.New("[ERROR] number of users per room is invalid, should be positive")
	} else if conf.SingleClientMode < 0 || conf.SingleClientMode > 1 {
		e = errors.New("[ERROR] single_client_mode is invalid, should be 0 or 1")
	} else if conf.ParallelMode < 0 || conf.ParallelMode > 2 {
		e = errors.New("[ERROR] parallel_mode is invalid, should be 0, 1, 2")
	} else if conf.WsReqConcurrency <= 0 {
		e = errors.New("[ERROR] ws_request_speed_number is invalid, should be positive")
	} else if conf.StartTimeRoom != "" {
		e = checkTime(conf.StartTimeRoom)
	} else if conf.StartTimeUser != "" {
		e = checkTime(conf.StartTimeUser)
	} else if conf.ParallelMode == 2 && conf.OnlineTime <= 0 {
		e = errors.New("[ERROR] ws_online_duration_in_second is invalid, should be positive")
	}
	return e
}

func processExtraHttpData(rm *roomManager, conf *Config) {
	if rm == nil || conf == nil {
		return
	}
	c := conf.CreateRoomExtraData.(map[string]interface{})

	rm.createRoomExtraData = make(map[string]string)
	for k, v := range c {
		switch v.(type) {
		case string:
			rm.createRoomExtraData[k] = v.(string)
		case float64:
			rm.createRoomExtraData[k] = fmt.Sprintf("%d", int(v.(float64)))
		default:
		}
	}

	h := conf.HttpHeaderExtra.(map[string]interface{})

	rm.httpHeaders = make(map[string]string)
	for k, v := range h {
		switch v.(type) {
		case string:
			rm.httpHeaders[k] = v.(string)
		default:
		}
	}
}

type Config struct {
	Host                string      `json:"host"`
	Room                int         `json:"room"`
	User                int         `json:"user"`
	Len                 int         `json:"msg_len"`
	Freq                int         `json:"msg_frequency"`
	RandomMsg           int         `json:"msg_random_send"`
	LogOutput           string      `json:"log_output"`
	AppID               string      `json:"app_id"`
	HttpTimeOut         int         `json:"http_timeout"`
	WSTimeOut           int         `json:"websocket_timeout"`
	StartTimeRoom       string      `json:"start_time_room"`
	StartTimeUser       string      `json:"start_time_user"`
	SingleClientMode    int         `json:"single_client_mode"`
	ParallelMode        int         `json:"parallel_mode"`
	WsReqConcurrency    int         `json:"ws_request_speed_number"`
	OnlineTime          int         `json:"room_expiration_time"`
	SDKVersion          string      `json:"sdk_version"`
	CreateRoomExtraData interface{} `json:"CreateRoomExtraData"` // the extra data which post to the server
	HttpHeaderExtra     interface{} `json:"http_header"`
}
