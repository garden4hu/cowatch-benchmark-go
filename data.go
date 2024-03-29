package main

import (
	"context"
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

	msgPool *sync.Pool

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
	name               string      // uuid of userInfo
	sid                string      // correspond with name ns
	uid                int         // digital id
	lock               *sync.Mutex // lock for writing
	connected          bool
	readyForMsg        bool
	connectionDuration time.Duration
	hostCoWatch        bool // only the userInfo who create the room can be the host
	expireTimer        *time.Ticker
	msgPool            *sync.Pool
	msgCtx             *context.Context
	msgCancelFunc      context.CancelFunc

	room *roomUnit

	messageTimer *time.Timer
	// for debug
	lastPing              time.Time
	pingIntervalStartTime time.Duration
	wsReqTimeOH           time.Duration // websocket request time over head
	wsPrologTimeOH        time.Duration // websocket prolog before clock sync time over head
	id                    int
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

type VideoAdvanced struct {
	PresentationDelay int `json:"presentationDelay"`
}
type VideoLink struct {
	Uri  string `json:"uri"`
	Type string `json:"type"`
}

type Video struct {
	Advanced VideoAdvanced `json:"advanced"`
	Links    []VideoLink   `json:"links"`
}

type ContentInfo struct {
	Videos Video `json:"video"`
}
