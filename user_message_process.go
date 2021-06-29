package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
)

// processMessage
func processMessage(conn *websocket.Conn, room *roomUnit, user *userInfo, ch chan bool, ctx context.Context) {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("websocket read:", err)
			return
		}
		// log.Printf("recv: %s", message)
		if err := processMsg(conn, message, user, room); err != nil {
			log.Println("[ERR] processMsg fatal err, goroutine will exit")
			ch <- true // 主动退出
			return
		}
		select {
		case <-ctx.Done():
			{
				// 上层取消
				log.Println("ctx 上层取消")
				return
			}
		default:
		}
	}
}

func processMsg(conn *websocket.Conn, b []byte, p *userInfo, room *roomUnit) error {
	b = bytes.TrimPrefix(b, []byte(" "))
	b = bytes.TrimSuffix(b, []byte(" "))
	if len(b) < 2 {
		return nil
	}
	msgType, _ := strconv.Atoi(string(b[0]))
	msgSubType, _ := strconv.Atoi(string(b[1]))
	if msgType == 4 {
		switch msgSubType {
		case 2:
			// process type == 2
			// _ = processNormalTypeMsg(conn, b, p, room)
			msg := string(b)
			if strings.Contains(msg, "REC:chatInit") {
				p.readyForMsg = true
			}
			break
		case 0:
			// 40, doesn't contain useful information currently.
			if len(b) > 2 {
				msg := "42/" + room.roomName + "," + "[\"CMD:name\",\"" + "tempUserName" + "\"]"
				p.lw.Lock()
				if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
					log.Println("[ERR] ws failed to send users name")
					return err
				}
				p.lw.Unlock()
			}
			break
		case 4:
			log.Println("[ERR] try to access an invalid room, raw msg:", string(b))
			return errors.New("[ERR] try to access an invalid room")
		default:
			log.Println("[WARN] received an unknown message, raw msg:", string(b))
			break
		}
	} else if msgType == 0 {
		userInfo := new(requestedUserInfo)
		err := json.Unmarshal(b[1:], userInfo)
		if err != nil {
			log.Println("failed to parsed the first message, raw msg:", string(b))
			return err
		}
		p.sid = userInfo.Sid
		room.pingInterval = userInfo.PingInterval
		room.pingTimeout = userInfo.PingTimeOut
		msgParam := "uid=" + strconv.Itoa(p.uid) + "&" + "name=" + p.name + "&version=" + room.sdkVersion
		//v := url.Values{}
		//v.Add("uid", strconv.Itoa(p.uid))
		//v.Add("name", p.name)
		//v.Add("version", room.sdkVersion)
		// message format : "40/C1LWsXh4jxXsfyK6MQSt?Sid=1174252488&name=ddd0&version=1.0.0-7289-integration-b2a92020,"
		msg := "40/" + room.roomName + "?" + msgParam + ","
		p.lw.Lock()
		if err = conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
			log.Println("[ERR] ws failed to send join room msg")
			return err
		}
		log.Println("send uuid mag :", msg)
		p.lw.Unlock()
		return nil
	} else if msgType == 3 {
		// pong message omit it
	} else {
		log.Println("received a msg will not be processed, start with :", b[0])
	}
	return nil
}

// processMessage message with prefix 42
//func processNormalTypeMsg(conn *websocket.Conn,b []byte, p *userInfo, room *roomUnit) error{
//	coreMsg := func() []byte {
//		i := 0
//		j := len(b)
//		for ; i < len(b); i++ {
//			if b[i] == '[' {
//				i++
//				break
//			}
//		}
//		for ; j >0; j-- {
//			if b[j] == ']' {
//				break
//			}
//		}
//		return b[i:j+1]
//	}()
//	var obj interface{}
//	if err := json.Unmarshal(coreMsg, &obj); err != nil{
//		log.Println("[ERR] failed to parse 42 msg, raw mag:",string(coreMsg))
//		return err
//	}
//	for k, v := range obj.(map[string]interface{}) {
//		switch k {
//		case "REC:nameMap":
//			// format: "/room_id#user_sid":"User_name"
//			break
//		case "REC:chatinit":
//			// we don't need to save these chat messages or even parse then.
//			break
//		case "REC:roster":
//			// format: "roomName":"/room_id#user_sid"
//			// same with nameMap. Do not need parse
//			switch v := v.(type) {
//			case []interface{}:
//				log.Println(k, "(array):")
//				for i, u := range v {
//					log.Println("    ", i, u)
//				}
//			default:
//				log.Println(k, v, "(unknown)")
//			}
//			break
//		case "REC:rtcToken":
//			// rtc token for audio/video
//			break
//		default:
//			// some message should be omitted.
//			break
//		}
//	}
//	return nil
//}
