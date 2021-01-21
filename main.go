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
)

var config = flag.String("c", "", "configure: configure file")
var addr = flag.String("host", "https://localhost:80", "host: address of coWatch server. schema://host")
var room = flag.Int("room", 1024, "room size: number of room to create")
var user = flag.Int("user", 1000, "user size: maximum number of user in room")
var msgLen = flag.Int("len", 1024, "message length: size of a message")
var frequency = flag.Int("freq", 1000, "frequency of sending message")
var logSwitch = flag.Int("log", 0, "log enable:1, disable:0")
var remoteConfig = flag.String("cr", "", "remote configure: remote configure. http(s)")
var httpReqTimeOut = flag.Int("t", 25, "create room timeout: http request timeout")
var startTimeCreatingRoom = flag.String("sr", "", "start time for creating room, following RFC3339. For example: 2017-12-08T00:08:00.00+08:00 . If not set, start instant.")
var startTimeCreatingUser = flag.String("su", "", "start time for creating user, following RFC3339. For example: 2017-12-08T00:08:00.00+08:00 . If not set, start instant.")

func Init() {
	flag.Parse()
}

func main() {
	Init()
	// process args
	var configure *Config
	if len(*config) > 0 {
		conf, err := readConfigure(*config)
		if err != nil {
			fmt.Println("[error] Failed to read configure, check your path or content of configure file, program will be exited.")
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
			fmt.Println("[error]  Program failed to download configure file from remote source, will be exited.")
			return
		}
		defer resp.Body.Close()
		b, _ := ioutil.ReadAll(resp.Body)
		var conf Config
		if err := json.Unmarshal(b, &conf); err != nil {
			fmt.Println("[error] Failed to read configure, check your path or content of configure file, program will be exited.")
			return
		}
		configure = &conf
	} else {
		configure = &Config{Host: *addr, Room: *room, User: *user, Len: *msgLen, Freq: *frequency, Log: *logSwitch, TimeOut: *httpReqTimeOut, StartTimeRoom: *startTimeCreatingRoom, StartTimeUser: *startTimeCreatingUser}
	}

	_, _ = fmt.Fprintf(os.Stdout, "Your Information:\n server address:\t %s\n number of room:\t %d\n users per room:\t %d\n size of message:\t %d\n frequency of messages:\t %d\n log enable:\t %d\n", configure.Host, configure.Room, configure.User, configure.Len, configure.Freq, configure.Log)
	if configure.Log == 0 {
		log.SetFlags(0)
		log.SetOutput(ioutil.Discard)
	}

	// register system interrupt
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	_ = run(configure)
	for {
		// create room
		select {
		case <-interrupt:
			log.Println("interrupt")
			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
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

type Config struct {
	Host          string `json:"host"`
	Room          int    `json:"room"`
	User          int    `json:"user"`
	Len           int    `json:"msg_len"`
	Freq          int    `json:"msg_frequency"`
	Log           int    `json:"log_enable"`
	TimeOut       int    `json:"timeout"`
	StartTimeRoom string `json:"start_time_room"`
	StartTimeUser string `json:"start_time_user"`
}
