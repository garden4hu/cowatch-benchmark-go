package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	cb "github.com/garden4hu/cowatchbenchmark"
)

var config = flag.String("c", "", "[Mandatory] configure: configure file")
var addr = flag.String("h", "http://localhost:80", "[Mandatory] host: address of coWatch server. schema://host")
var room = flag.Int("r", 1024, "room size: number of room to create")
var user = flag.Int("u", 1000, "user size: maximum number of user in room")
var msgLen = flag.Int("l", 1024, "message length: size of a message")
var frequency = flag.Int("f", 1000, "frequency: frequency of sending message")
var logSwitch = flag.Int("v", 0, "verbose log enable:1, disable(default):0")
var remoteConfig = flag.String("cr", "", "[Mandatory] remote configure: remote configure. No coexistence with -c")
var httpReqTimeOut = flag.Int("th", 25, "http timeout(1~60s): http request timeout for create room")
var wsReqTimeOut = flag.Int("tw", 45, "websocket timeout(1~60s): websocket request timeout for create user")
var startTimeCreatingRoom = flag.String("rs", "", "[Mandatory] start time for creating room: following RFC3339. For example: 2017-12-08T00:08:00.00+08:00")
var startTimeCreatingUser = flag.String("us", "", "[Mandatory] start time for creating user: following RFC3339. For example: 2017-12-08T00:08:00.00+08:00")

func Init() {
	flag.Parse()
}

var roomManager *cb.RoomManager

func main() {
	Init()
	color.Unset()
	color.Set(color.Bold, color.FgHiRed)
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
		configure = &Config{Host: *addr, Room: *room, User: *user, Len: *msgLen, Freq: *frequency, Log: *logSwitch, HttpTimeOut: *httpReqTimeOut, WSTimeOut: *wsReqTimeOut, StartTimeRoom: *startTimeCreatingRoom, StartTimeUser: *startTimeCreatingUser}
	}
	color.Unset()

	_, _ = fmt.Fprintf(os.Stdout, "\ninput info:\n server address:\t %s\n number of room:\t %d\n users per room:\t %d\n message length:\t %d\n message frequency:\t %d\n log enable:\t %d\n\n", configure.Host, configure.Room, configure.User, configure.Len, configure.Freq, configure.Log)
	if configure.Log == 0 {
		log.SetFlags(0)
		log.SetOutput(ioutil.Discard)
	}

	// register system interrupt
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	if err := getRooms(configure); err != nil {
		color.Set(color.FgRed)
		fmt.Println(err)
		color.Unset()
		return
	}
	if err := getUsers(configure); err != nil {
		color.Set(color.FgRed)
		fmt.Println(err)
		color.Unset()
		return
	}
	// ticker used for get statistics information
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		// create room
		select {
		case <-interrupt:
			log.Println("interrupt")
			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			return
		case <-ticker.C:
			go printLogMessage(roomManager, configure)
			ticker.Reset(5 * time.Second)
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

type Config struct {
	Host          string `json:"host"`
	Room          int    `json:"room"`
	User          int    `json:"user"`
	Len           int    `json:"msg_len"`
	Freq          int    `json:"msg_frequency"`
	Log           int    `json:"log_enable"`
	HttpTimeOut   int    `json:"http_timeout"`
	WSTimeOut     int    `json:"websocket_timeout"`
	StartTimeRoom string `json:"start_time_room"`
	StartTimeUser string `json:"start_time_user"`
}
