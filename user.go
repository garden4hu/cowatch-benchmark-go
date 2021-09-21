package main

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// newUser construct the user
func newUser() *userInfo {
	return &userInfo{name: generateUserName(8), uid: getHostId(), hostCoWatch: false, connected: false, readyForMsg: false, expireTimer: time.NewTicker(24 * time.Hour)}
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
				return newUser()
			}
		}()
		go u.joinRoom(ctx, p, p.rm.parallelRequest, start, &wg2)
	}
	wg2.Wait()
	if p.rm.parallelRequest == false {
		time.Sleep(10 * time.Millisecond)
	}
}

// joinRoom join the room on the websocket server
func (user *userInfo) joinRoom(ctx context.Context, r *roomUnit, parallel bool, start chan struct{}, wg *sync.WaitGroup) {
	// if request users concurrently, goroutine should be waited
	if wg != nil {
		wg.Done() // create goroutine done
	}
	if parallel {
		if start != nil {
			<-start // waiting for starting
		}
	}
	startJoin := time.Now()
	conn, err := wsConnect(ctx, r, user)
	if err != nil {
		log.Errorln("failed to create websocket connection, err = ", err)
		return
	}
	// add user to roomUnit
	r.muxUsers.Lock()
	r.users = append(r.users, user) // add user to room
	r.muxUsers.Unlock()
	r.rm.notifyUserAdd <- 1 // 通知新增用户

	user.connectionDuration = time.Since(startJoin)
	defer conn.Close()
	user.connected = true

	defer func() {
		r.rm.notifyUserAdd <- -1
	}() // 通知用户下线

	done := make(chan bool)
	defer close(done)

	receivedData := make(chan []byte)
	defer close(receivedData)
	// starting a new goroutine for receiveMessage the websocket message
	go user.receiveMessage(ctx, conn, done, receivedData)

	//pingTicker := time.NewTicker(time.Millisecond * time.Duration(r.pingInterval))
	//log.Println("ping ticker duration:", r.pingInterval)
	//defer pingTicker.Stop()

	// 对于测试环境而言，host 发送的 sync 信息频次较高，故对于 host userInfo，需要考虑其发送频率
	// 而对于 Guests, 其 websocket 消息内容更多为 text，ping/pong，这些消息频次较低
	sendMsgTicker := time.NewTicker(r.msgSendingInternal)
	defer sendMsgTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			// need to reconnect
			_ = conn.Close()
			conn, err = wsConnect(ctx, r, user)
			if err != nil {
				log.Errorln("failed to reconnect to ws server, err = ", err)
				return
			} else {
				go user.receiveMessage(ctx, conn, done, receivedData)
			}
			break
			//case _ = <-pingTicker.C:
			//	// reset pingTicker and send ping
			//	if conn != nil {
			//		user.lw.Lock()
			//		err := conn.WriteMessage(websocket.TextMessage, []byte("2"))
			//		user.lw.Unlock()
			//		if err != nil {
			//			log.Println("write:", err)
			//		}
			//	}
			//	pingTicker.Reset(time.Millisecond * time.Duration(r.pingInterval))

		case <-user.expireTimer.C:
			return
		case _ = <-sendMsgTicker.C:
			if user.hostCoWatch {
				// 在测试环境中，由于用户的 text 的信息数量可以忽略，故此处只允许 host 发送消息到服务器
				if user.connected {
					msg := generateMessage(r)
					if conn != nil {
						_ = conn.WriteMessage(websocket.TextMessage, msg)
					}
					sendMsgTicker.Reset(r.msgSendingInternal)
				}
			}
		case rd := <-receivedData:
			log.Debugln("received ws message:", string(rd))
			err := user.sendResponse(conn, rd, r)
			if err != nil {
				return
			}
		default:

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
		break
	case "https":
		u.Scheme = "wss"
		break
	default:
		u.Scheme = "wss"
		break
	}
	dialer := &websocket.Dialer{
		Proxy:             http.ProxyFromEnvironment,
		HandshakeTimeout:  r.wsTimeout,
		EnableCompression: true,
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
