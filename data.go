package main

import (
	"sync"
	"time"
)

type roomUnit struct {
	address      string        // server+port
	schema       string        // http(s)
	ns           string        // room name
	roomId       int           // room ID
	password     string        // url param
	httpTimeout  time.Duration // timeout for http request
	wsTimeout    time.Duration // timeout for ws request
	pingInterval int           // ws keep-alive
	pingTimeout  int           // ws ping timeout
	rtcToken     string        // rtc token for rtc continuation
	muxUsers     sync.Mutex
	users        []*userInfo // valid users in a room
	appId        string      // application id for rtc
	expireTime   int         // expire duration for room living
	sdkVersion   string      // sdk version

	condMutex *sync.Mutex // used for conditional waiting
	cond      *sync.Cond

	// for statistics
	connectionDuration time.Duration

	rm *roomManager

	// for internal usage
	chanStop           chan bool
	wg                 sync.WaitGroup
	start              bool          // flag of starting to concurrent usersConnection
	usersCap           int           // users cap in this room
	usersOnline        int           // online users
	msgLength          int           // length of message
	msgSendingInternal time.Duration // Microsecond as the unit
}

type userInfo struct {
	name               string     // uuid of userInfo
	sid                string     // correspond with name ns
	uid                int        // digital id
	lw                 sync.Mutex // lock for writing
	connected          bool
	readyForMsg        bool
	connectionDuration time.Duration
	hostCoWatch        bool // only the userInfo who create the room can be the host
	expireTimer        *time.Ticker
}

type requestedUserInfo struct {
	Sid          string `json:"sid"`
	Upgrades     []int  `json:"upgrades"`
	PingInterval int    `json:"pingInterval"`
	PingTimeOut  int    `json:"pingTimeout"`
}

type roomInfo struct {
	Name string `json:"name"`
}
