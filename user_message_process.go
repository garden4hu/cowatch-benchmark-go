package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/gorilla/websocket"
	"strconv"
	"strings"
	"time"
)

// receiveMessage
func (user *userInfo) receiveMessage(ctx context.Context, conn *websocket.Conn, ch chan bool, rdata chan []byte) {
	for {
		if conn != nil {
			// Read data
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Errorln("websocket read:", err)
				select {
				case <-ctx.Done():
					return
				default:
					ch <- true
				}
			} else {
				select {
				case <-ctx.Done():
					return
				default:
					b := make([]byte, len(message))
					copy(b, message)
					rdata <- b
				}
			}
		}
	}
}

func (user *userInfo) sendResponse(conn *websocket.Conn, b []byte, room *roomUnit) (e error) {
	log.Debugln("user processing response message: ", string(b))
	b = bytes.TrimPrefix(b, []byte(" "))
	b = bytes.TrimSuffix(b, []byte(" "))
	if len(b) < 1 {
		return nil
	}
	log.Debugln(string(b))
	t, _ := strconv.Atoi(string(b[0]))

	v := engineIOV4Type(t)
	switch v {
	case engineTypeOPEN:
		if e = user.onEngineOpen(b, room, conn); e != nil {
			return e
		}
	case engineTypePING: // ping of engineIO
		msg := []byte("3")
		log.Debugln("received ping 2, sending pong 3")
		if e = conn.WriteMessage(websocket.TextMessage, msg); e != nil {
			log.Errorln("[ERR] ws failed to send pong")
			return e
		}
	case engineTypeMESSAGE:
		if e = user.processSocketIO(b[1:], room, conn); e != nil {
			log.Errorln("[ERR] ws failed to send join room msg")
			return e
		}

	}

	return e
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

func warpString(s string) string {
	return "\"" + s + "\""
}

func generateMsgBody(msgCMD string, jsonBody string) string {
	b, e := json.Marshal(warpString(msgCMD) + "," + jsonBody)
	if e != nil {
		return ""
	}
	return "[" + string(b) + "]"
}

func (user *userInfo) generateConnectAndDisconnectMessage(n socketIOV4Type, ns string) []byte {
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
func (user *userInfo) generateEventMessage(ns string, id *int, msgCMD string, body string) []byte {
	if ns != "" {
		ns = "/" + ns
	}
	tID := ""
	if id != nil {
		tID = strconv.Itoa(*id)
	}
	return []byte("42" + ns + "," + tID + generateMsgBody(msgCMD, body))

}

func (user *userInfo) generateACKMessage(ns string, id int, data string) []byte {
	if ns != "" {
		ns = "/" + ns
	}
	data = "" // data doesn't support now
	return []byte("43" + ns + "," + strconv.Itoa(id) + data)
}

func (user *userInfo) generateERRORMessage(ns string, data string) []byte {
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

func (user *userInfo) processSocketIO(b []byte, room *roomUnit, conn *websocket.Conn) error {
	t := socketIOV4Type(b[0])
	switch t {
	case socketTypeCONNECT: // socket.io CONNECT
		user.onSocketIOConnect(b)
	case socketTypeDISCONNECT:
		return errors.New("error: socket finish")
	case socketTypeEVENT:
		return user.onSocketIOEvent(b, room, conn)
	case socketTypeACK:
	case socketTypeBINARYACK:
		// unsupported
	case socketTypeBINARYEVENT:
		// unsupported
	default:
	}
	return nil
}

func (user *userInfo) onEngineOpen(b []byte, room *roomUnit, conn *websocket.Conn) error {
	log.Debugln("Received EngineIO Open event")
	userInfo := new(requestedUserInfo)
	err := json.Unmarshal(b[1:], userInfo)
	if err != nil {
		log.Errorln("failed to parsed the first message, raw msg:", string(b))
		return err
	}
	user.sid = userInfo.Sid
	room.pingInterval = userInfo.PingInterval
	room.pingTimeout = userInfo.PingTimeOut

	msg := user.generateConnectAndDisconnectMessage(socketTypeCONNECT, room.ns)
	log.Debugln("pong sent")
	if err = conn.WriteMessage(websocket.TextMessage, msg); err != nil {
		log.Errorln("[ERR] ws failed to send join room msg")
		return err
	}
	return nil
}

func (user *userInfo) onSocketIOConnect(b []byte) {
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

func (user *userInfo) onSocketIOEvent(b []byte, room *roomUnit, conn *websocket.Conn) (e error) {
	data := getData(b)
	vs := strings.SplitAfter(string(data), ",")
	cmd := vs[0]
	switch cmd {
	case "expireTime":
		et := new(expireTime)
		if e := json.Unmarshal([]byte(vs[1]), et); e == nil {
			user.expireTimer.Reset(time.Duration(et.LeftTime) * time.Minute)
		}
	case "REC:roomInit":
		// get hostIid, lock status and rtcToken
	case "REC:roster":
		// get user list. Now join room ok
		user.connected = true
		ct := new(clientTime)
		for i := 0; i < 11; i++ {
			ct.ClientTime = time.Now().UnixMilli()
			b, e := json.Marshal(ct)
			if e != nil {
				time.Sleep(20 * time.Millisecond)
				continue
			}
			msg := user.generateEventMessage(room.ns, nil, "REC:clockSync", string(b))
			e = conn.WriteMessage(websocket.TextMessage, msg)
			if e != nil {
				return e
			}
		}
	case "REC:userAdded":
		// received user added info
	case "REC:clockSync":
		// received clockSync
		ct := new(clientTime)
		ct.ClientTime = time.Now().UnixMilli()
		b, e := json.Marshal(ct)
		if e != nil {
			time.Sleep(20 * time.Millisecond)
			break
		}
		msg := user.generateEventMessage(room.ns, nil, "REC:clockSync", string(b))
		e = conn.WriteMessage(websocket.TextMessage, msg)
		if e != nil {
			return e
		}
	default:

	}
	return e
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
