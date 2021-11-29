package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/gorilla/websocket"
	"strconv"
	"strings"
	"time"
)

// processMessageWorker
func processMessageWorker(user *userInfo, conn *websocket.Conn) {
	defer logIn.Debugln("goID:", user.id, " receiving goroutine exit")
	defer func() {
		time.Sleep(100 * time.Millisecond) // waiting writing goroutine exit
		_ = conn.Close()
		user.room.rm.notifyUserAdd <- -1 // user offline
		user.connected = false
		conn = nil
	}()
	timeStampPlayBackMessage := time.Now()
	for {
		if user.hostCoWatch {
			if time.Since(timeStampPlayBackMessage) > 10*time.Minute {
				pbStr := "42/" + user.room.ns + ",[\"CMD:contentInfo\"," + user.room.rm.playBackContent + "]"
				pb := []byte(pbStr)
				user.lock.Lock()
				err := conn.WriteMessage(websocket.TextMessage, pb)
				user.lock.Unlock()
				if err != nil {
					logA.Errorln("goID:", user.id, " failed to write ws connection playback message")
					return
				}
				timeStampPlayBackMessage = time.Now()
			}
		}
		// Read data
		_, message, err := conn.ReadMessage()
		if err != nil {
			logIn.Warnln("goID:", user.id, " websocket read message error:", err)
			return
		} else {
			reply, err := processResponse(user, message, user.room)
			if err != nil {
				logA.Errorln("goID:", user.id, " failed to process message: ", string(message))
			}
			if len(reply) > 0 {
				user.lock.Lock()
				err := conn.WriteMessage(websocket.TextMessage, reply)
				user.lock.Unlock()
				if err != nil {
					logA.Errorln("goID:", user.id, " failed to write ws connection")
					return
				}
			}
		}
	}
}

func processResponse(user *userInfo, b []byte, room *roomUnit) (msg []byte, err error) {
	log.Debugln("goID:", user.id, " client incoming message: ", string(b))
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return nil, nil
	}
	t, _ := strconv.Atoi(string(b[0]))
	v := engineIOV4Type(t)
	switch v {
	case engineTypeOPEN:
		msg, err = onEngineOpen(user, b, room)
		if err != nil || len(msg) == 0 {
			logA.Errorln("goID:", user.id, " replay generate error, msg")
		}
	case engineTypePING: // ping of engineIO
		logIn.Debugln("goID:", user.id, " ping_pong")
		msg = []byte("3")
		return msg, nil
	case engineTypeMESSAGE:
		msg, err = processSocketIO(user, b[1:], room)
	default:
	}
	return
}

func processSocketIO(user *userInfo, b []byte, room *roomUnit) (msg []byte, err error) {
	t, _ := strconv.Atoi(string(b[0]))
	switch socketIOV4Type(t) {
	case socketTypeCONNECT: // socket.io CONNECT
		onSocketIOConnect(user, b)
	case socketTypeDISCONNECT:
		err = errors.New("error: socket finish")
	case socketTypeEVENT:
		msg, err = onSocketIOEvent(user, b, room)
	case socketTypeACK:
	case socketTypeBINARYACK:
		// unsupported
	case socketTypeBINARYEVENT:
		// unsupported
	default:
	}
	return msg, err
}

func onEngineOpen(user *userInfo, b []byte, room *roomUnit) (msg []byte, err error) {
	// reset the pint ticker
	log.Debugln("goID:", user.id, " Received EngineIO Open event")
	u := new(requestedUserInfo)
	err = json.Unmarshal(b[1:], u)
	if err != nil {
		logA.Errorln("goID:", user.id, " failed to parsed the first message, raw msg:", string(b))
		return nil, err
	}
	user.sid = u.Sid
	room.pingInterval = u.PingInterval
	logIn.Debugln("goID:", user.id, " ping interval is :", room.pingInterval, " ping timeout  is :", u.PingInterval)
	room.pingTimeout = u.PingTimeOut

	msg = generateConnectAndDisconnectMessage(socketTypeCONNECT, room.ns)
	log.Debugln("goID:", user.id, " process EngineIO Open event: response msg:", string(msg))

	// notify
	if err == nil {
		room.rm.notifyUserPingOK <- 1
	}
	return msg, err
}

func onSocketIOConnect(user *userInfo, b []byte) {
	i := 0
	for ; i < len(b); i++ {
		if b[i] == 0x2C {
			i++
			break
		}
	}
	if i >= len(b) {
		return
	}
	b = b[i:]
	sid := make(map[string]string)
	if e := json.Unmarshal(b, &sid); e != nil {
		return
	}
	user.sid = sid["sid"]
}

type expireTime struct {
	LeftTime int `json:"leftTime"`
}

type clientTime struct {
	ClientTime int64 `json:"clientTime"`
}

func onSocketIOEvent(user *userInfo, b []byte, room *roomUnit) (msg []byte, err error) {
	data := getData(b)
	vs := strings.Split(string(data), ",")
	var cmd string
	err = json.Unmarshal([]byte(vs[0]), &cmd)
	if err != nil {
		logIn.Println("goID:", user.id, " failed to parse response json")
		return nil, err
	}
	switch cmd {
	case "expireTime":
		et := new(expireTime)
		if e := json.Unmarshal([]byte(vs[1]), et); e == nil {
			user.expireTimer.Reset(time.Duration(et.LeftTime) * time.Minute)
		}
	case "REC:roomInit":
		// get hostIid, lock status and rtcToken
		// get user list. Now join room ok
		room.rm.notifyUserAdd <- 1 // user online
		user.connected = true
		logIn.Debugln("goID:", user.id, " get roster, connected")
		// random distribution
		time.Sleep(time.Duration(time.Millisecond) * time.Millisecond)
		msg = generateClockSyncMessage(user, room)
	case "REC:roster":
		// user list was merged to roomInit.
	case "REC:userAdded":
		// received user added info
	case "REC:clockSync":
		// received clockSync
		msg = generateClockSyncMessage(user, room)
	default:

	}
	return msg, err
}
func getData(b []byte) []byte {
	i := 0
	for ; i < len(b); i++ {
		if b[i] == 0x5B {
			i++
			break
		}
	}
	if i < len(b) {
		j := i
		for ; j < len(b); j++ {
			if b[j] == 0x5D {
				break
			}
		}
		if j < len(b) {
			return b[i:j]
		}
	}
	return nil
}

type chatMsg struct {
	Msg string `json:"msg"`
}

// generateMessage generate text message randomly for user
func generateMessage(r *roomUnit) []byte {
	cm := new(chatMsg)
	cm.Msg = randStringBytes(r.msgLength)
	b, _ := json.Marshal(cm)
	msg := "42/" + r.ns + ",[\"CMD:chat\"," + string(b) + "]"
	return []byte(msg)
}

func generateClockSyncMessage(user *userInfo, r *roomUnit) []byte {
	ct := new(clientTime)
	ct.ClientTime = time.Now().UnixMilli()
	buf, _ := json.Marshal(ct)
	return generateEventMessage(r.ns, nil, "REC:clockSync", buf)
}

type engineIOV4Type int8

const (
	engineTypeOPEN engineIOV4Type = iota
	engineTypeCLOSE
	engineTypePING
	engineTypePONG
	engineTypeMESSAGE
	engineTypeUPGRADE
	engineTypeNOOP
)

func (n engineIOV4Type) toInt() int8 {
	switch n {
	case engineTypeOPEN:
		return 0
	case engineTypeCLOSE:
		return 1
	case engineTypePING:
		return 2
	case engineTypePONG:
		return 3
	case engineTypeMESSAGE:
		return 4
	case engineTypeUPGRADE:
		return 5
	case engineTypeNOOP:
		return 6
	}
	return 4
}

type socketIOV4Type int8

const (
	socketTypeCONNECT socketIOV4Type = iota
	socketTypeDISCONNECT
	socketTypeEVENT
	socketTypeACK
	socketTypeERROR
	socketTypeBINARYEVENT
	socketTypeBINARYACK
)

func (n socketIOV4Type) toInt() int8 {
	switch n {
	case socketTypeCONNECT:
		return 0
	case socketTypeDISCONNECT:
		return 1
	case socketTypeEVENT:
		return 2
	case socketTypeACK:
		return 3
	case socketTypeERROR:
		return 4
	case socketTypeBINARYEVENT:
		return 5
	case socketTypeBINARYACK:
		return 6
	}
	return 2
}

func generateMsgBody(msgCMD string, body []byte) string {
	b, e := json.Marshal(msgCMD)
	if e != nil {
		return ""
	}
	return "[" + string(b) + "," + string(body) + "]"
}

func generateConnectAndDisconnectMessage(n socketIOV4Type, ns string) []byte {
	log.Debugln("generate Open msg, ns: ", ns)
	if ns != "" {
		ns = "/" + ns
	}
	s := ""
	switch n {
	case socketTypeCONNECT:
		s = "40"
	case socketTypeDISCONNECT:
		s = "41"
	}
	return []byte(s + ns + ",")
}
func generateEventMessage(ns string, id *int, msgCMD string, body []byte) []byte {
	if ns != "" {
		ns = "/" + ns
	}
	tID := ""
	if id != nil {
		tID = strconv.Itoa(*id)
	}
	return []byte("42" + ns + "," + tID + generateMsgBody(msgCMD, body))

}

func generateACKMessage(ns string, id int, data string) []byte {
	if ns != "" {
		ns = "/" + ns
	}
	data = "" // data doesn't support now
	return []byte("43" + ns + "," + strconv.Itoa(id) + data)
}

func generateERRORMessage(ns string, data string) []byte {
	if ns != "" {
		ns = "/" + ns
	}
	b, e := json.Marshal(data)
	if e != nil {
		data = ""
	} else {
		data = string(b)
	}
	return []byte("44" + ns + "," + data)
}
