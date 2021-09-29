package main

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// newUser construct the user
func newUser(p *roomUnit) *userInfo {
	u := &userInfo{name: generateUserName(8), uid: getHostId(), hostCoWatch: false, connected: false, readyForMsg: false, expireTimer: time.NewTicker(24 * time.Hour), msgPool: p.msgPool}
	u.id = int(atomic.AddInt32(&gID, 1))
	return u
}

// usersConnection try to connect to the server and exchange message.
// param when is the time for requesting of websocket concurrently
// param mode is the mode for requesting. 0 means parallel and 1 means serial
func (p *roomUnit) usersConnection(start chan struct{}, ctx context.Context, wg *sync.WaitGroup) {
	if p.rm.parallelRequest {
		defer wg.Done()
	}
	// create users
	var wg2 sync.WaitGroup // 用于并发请求，确保所有的 goroutine 同时发起请求，而不会出现开始并发请求时，有的 goroutine 还没有构造好 ws 句柄
	for i := 1; i < p.usersCap; i++ {
		wg2.Add(1)
		u := func() *userInfo {
			if i == 0 {
				return p.users[0]
			} else {
				return newUser(p)
			}
		}()
		go u.joinRoom(ctx, p, p.rm.parallelRequest, start, &wg2)
		time.Sleep(500 * time.Millisecond) // avoid concurrent
	}
	wg2.Wait()
	if p.rm.parallelRequest {
		time.Sleep(600 * time.Millisecond)
	}
}

// joinRoom join the room on the websocket server
func (user *userInfo) joinRoom(ctx context.Context, r *roomUnit, parallel bool, start chan struct{}, wg *sync.WaitGroup) {
	defer log.Infoln("goroutine exist")

	defer func() {
		if user.pingTimer != nil {
			user.pingTimer.Stop()
		}
	}()
	// if request users concurrently, goroutine should be waited
	if wg != nil {
		wg.Done() // create goroutine done
	}
	if parallel {
		if start != nil {
			<-start // waiting for starting
		}
	}

	// add user to roomUnit
	r.muxUsers.Lock()
	r.users = append(r.users, user) // add user to room
	r.muxUsers.Unlock()

	messageCh := make(chan []byte)
	defer close(messageCh)
	user.pingTimer = time.NewTicker(60 * time.Second)
	startWS := func() (context.Context, error) {
		msgCtx, cancel := context.WithCancel(ctx)
		startJoin := time.Now()
		conn, err := wsConnect(msgCtx, r, user)
		if err == nil {
			user.wsReqTimeOH = time.Since(startJoin)
			logIn.Debugln("goID:", user.id, " ws_req_time(ms):", user.wsReqTimeOH.Milliseconds())
			go user.receiveMessage(conn, r, messageCh, cancel)
			go user.sendMessage(conn, r, messageCh, r.msgSendingInternal, cancel)
			user.connectionDuration = time.Since(startJoin)
			return msgCtx, err
		} else {
			log.Errorln("failed to launch ws")
			cancel()
			return nil, err
		}
	}

	// 对于测试环境而言，host 发送的 sync 信息频次较高，故对于 host userInfo，需要考虑其发送频率
	// 而对于 Guests, 其 websocket 消息内容更多为 text，ping/pong，这些消息频次较低
	wsCTX, err := startWS()
	if err != nil {
		return
	}
	sendMsgTicker := time.NewTicker(r.msgSendingInternal)
	defer sendMsgTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-wsCTX.Done():
			// try to reconnect
			r.rm.notifyUserAdd <- -1 // user offline
			user.connected = false
			i := 0
			for ; i < 1; i++ {
				wsCTX, err = startWS()
				if err != nil {
					continue
				} else {
					break
				}
			}
			if i == 3 {
				log.Errorln("failed to reconnect to websocket server")
				return
			}
			log.Infoln("reconnect websocket server successfully")
		case <-user.expireTimer.C:
			return

		}
	}
}

func wsConnect(ctx context.Context, r *roomUnit, p *userInfo) (*websocket.Conn, error) {
	// set ws/wss url param
	v := url.Values{}
	v.Add("uid", strconv.Itoa(p.uid))
	v.Add("name", p.name)
	v.Add("version", r.sdkVersion)
	v.Add("roomId", r.ns)
	v.Add("EIO", "4") // using socket.io V4
	v.Add("transport", "websocket")
	v.Add("hostname", "qa.visualon.com") // God knows why the hostname is set like this
	u := url.URL{Host: r.address, Path: "/socket.io/", ForceQuery: true, RawQuery: v.Encode()}

	switch r.schema {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	default:
		u.Scheme = "wss"
	}

	//pool := &sync.Pool{New: func() interface{} {
	//	s := new(poolData)
	//	return &s
	//}}
	dialer := &websocket.Dialer{
		Proxy:             http.ProxyFromEnvironment,
		HandshakeTimeout:  r.wsTimeout,
		EnableCompression: true,
		WriteBufferSize:   128, // because the longest size of message is 250, 128 is half of the largest size of message according to the guide of gorilla
		WriteBufferPool:   &sync.Pool{},
	}
	// set http->websocket header
	rq := http.Header{}
	rq.Add("Accept-Encoding", "gzip, deflate, br")
	rq.Add("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6")
	rq.Add("Cache-Control", "no-cache")
	// rq.Add("Connection","Upgrade")
	rq.Add("Pragma", "no-cache")
	for key, val := range r.rm.httpHeaders {
		rq.Add(key, val)
	}
	//rq.Add("Sec-WebSocket-Extensions","permessage-deflate; client_max_window_bits") // enable compress
	var conn *websocket.Conn
	var err error = nil
	for i := 0; i < 3; i++ {
		conn, _, err = dialer.DialContext(ctx, u.String(), rq)
		if err != nil {
			log.Errorln("websocket dialer error: ", err)
			if i < 3 {
				time.Sleep(50 * time.Millisecond)
				continue
			}
		} else {
			break
		}
	}
	if err != nil {
		log.Errorln("failed to dial websocket:", err)
		return nil, err
	}
	return conn, nil
}
