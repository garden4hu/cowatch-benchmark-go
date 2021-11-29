package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/fs"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
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
var logA = logrus.New()

var logIn = logrus.New()
var logOut = logrus.New()
var inlog *os.File
var outlog *os.File

var gID int32 = 0 // global number for generate user id

func init() {
	flag.Parse()
	onlineUser = 0
	analytics = new(statistic)
	webSocketRunningDuration = time.NewTicker(1 * time.Hour)
	initLog()
	exitFlag = make(chan bool)
}

var rm *roomManager
var onlineUser int
var analytics *statistic
var webSocketRunningDuration *time.Ticker

var onlineUserPingOK int

var exitFlag chan bool

func main() {

	// debug
	go func() {
		log.Println(http.ListenAndServe("localhost:16060", nil))
	}()

	defer func() {
		if inlog != os.Stdout && inlog != os.Stderr {
			inlog.Close()
		}
		if outlog != os.Stdout && outlog != os.Stderr {
			outlog.Close()
		}
	}()
	// process args
	var configure *Config
	if len(*config) > 0 {
		conf, err := readConfigure(*config)
		if err != nil {
			logA.Errorln("failed to read configure, check your path or content of configure file, program will be exited")
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
			logA.Errorln("[error]  Program failed to download configure file from remote source, will be exited")
			return
		}
		defer resp.Body.Close()
		b, _ := ioutil.ReadAll(resp.Body)
		var conf Config
		if err := json.Unmarshal(b, &conf); err != nil {
			logA.Errorln("failed to read configure file, check your path or make sure the content is ok, program will be exited now")
			return
		}
		configure = &conf
	} else {
		logA.Errorln("program doesn't support command, please set the configure file, it will be exited now")
		return
	}
	indent, err := json.MarshalIndent(configure, "", "\t")
	if err == nil {
		logA.Infoln("your configure is :\n", string(indent))
	}

	if e := checkConfigure(configure); e != nil {
		logA.Errorln(e.Error())
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
			logA.Warnln("interrupted by user")
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
		case t := <-rm.notifyUserPingOK:
			onlineUserPingOK += t
			break
		case <-webSocketRunningDuration.C:
			webSocketRunningDuration.Stop()
			logA.Warnln("The program exits at time")
			return
		case <-exitFlag:
			logA.Errorln("encounter an error, and exit")
			return
		default:
			break
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

func checkConfigure(conf *Config) (e error) {
	checkTime := func(tim string) (err error) {
		_, err = time.Parse(time.RFC3339, tim)
		if err != nil {
			err = errors.New("time string is invalid. A valid example is looks like: 2017-12-08T00:08:00.00+08:00")
		}
		return err
	}
	if conf.Address == "" {
		e = errors.New("[ERROR] Address invalid")
	} else if conf.Rooms <= 0 {
		conf.Rooms = 1
	} else if conf.UsersPerRoom <= 0 {
		conf.UsersPerRoom = 1
	} else if conf.Freq <= 0 {
		conf.Freq = 10
	} else if (60*1000)/conf.Freq < 1 {
		conf.Freq = 10
	} else if conf.SingleClientMode < 0 || conf.SingleClientMode > 1 {
		conf.SingleClientMode = 1
	} else if conf.ParallelMode < 0 || conf.ParallelMode > 2 {
		conf.ParallelMode = 0
	} else if conf.WsReqConcurrency <= 0 {
		conf.WsReqConcurrency = 1
	} else if conf.StartTimeRoom != "" {
		e = checkTime(conf.StartTimeRoom)
	} else if conf.StartTimeUser != "" {
		e = checkTime(conf.StartTimeUser)
	} else if conf.ParallelMode == 2 && conf.OnlineTime <= 0 {
		conf.OnlineTime = 300
	} else if conf.AppID == "" {
		e = errors.New("invalid app_id")
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

func initLog() {
	logF.Formatter = &logrus.JSONFormatter{}
	log.Formatter = &logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	}
	logA.Formatter = &logrus.TextFormatter{
		ForceColors: true,
	}

	logIn.Formatter = &logrus.TextFormatter{
		FullTimestamp: true,
	}
	logOut.Formatter = &logrus.TextFormatter{
		FullTimestamp: true,
	}
	workDir := filepath.Dir(os.Args[0])
	infile := filepath.Join(workDir, "in.log")
	if fs.ValidPath(infile) {
		_ = os.Remove(infile)
	}
	outfile := filepath.Join(workDir, "out.log")
	if fs.ValidPath(outfile) {
		_ = os.Remove(outfile)
	}
	inHandler, err := os.Create(infile)
	if err == nil {
		inlog = inHandler
	} else {
		inHandler = os.Stdout
	}
	outHandler, err := os.Create(outfile)
	if err == nil {
		outlog = outHandler
	} else {
		outHandler = os.Stdout
	}

	logIn.SetOutput(inHandler)
	logOut.SetOutput(outHandler)
	logIn.SetLevel(logrus.DebugLevel)
	logOut.SetLevel(logrus.DebugLevel)

	log.SetReportCaller(true)
	log.SetOutput(os.Stderr)
	logF.SetOutput(os.Stdout)
	logA.SetOutput(os.Stdout)
	log.SetLevel(logrus.InfoLevel)
	logF.SetLevel(logrus.InfoLevel)
	logF.SetLevel(logrus.InfoLevel)
	if *verbose == true {
		log.SetLevel(logrus.DebugLevel)
		logF.SetLevel(logrus.DebugLevel)
	}
}

type Config struct {
	Address             string      `json:"address"`
	Rooms               int         `json:"rooms"`
	UsersPerRoom        int         `json:"users_per_room"`
	Len                 int         `json:"message_length"`
	Freq                int         `json:"message_frequency"`
	AppID               string      `json:"app_id"`
	HttpTimeOut         int         `json:"http_request_timeout"`
	WSTimeOut           int         `json:"websocket_request_timeout"`
	StartTimeRoom       string      `json:"start_time_for_create_rooms"`
	StartTimeUser       string      `json:"start_time_for_create_users"`
	SingleClientMode    int         `json:"single_client_mode"`
	ParallelMode        int         `json:"parallel_mode"`
	WsReqConcurrency    int         `json:"ws_request_speed_number_for_mode_2"`
	OnlineTime          int         `json:"room_expiration_time_in_second"`
	SDKVersion          string      `json:"sdk_version"`
	CreateRoomExtraData interface{} `json:"createRoomExtraData"` // the extra data which post to the server
	HttpHeaderExtra     interface{} `json:"http_header"`
	Video               Video       `json:"video"`
}
