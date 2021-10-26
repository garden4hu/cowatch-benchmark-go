package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// newRoom return a roomUnit object
func newRoom(host string, httpTimeout, wsTimeout time.Duration, maximumUsers, msgLength, frequency int, appId string, rm *roomManager) *roomUnit {
	room := &roomUnit{usersCap: maximumUsers, msgLength: msgLength, msgSendingInternal: time.Microsecond * 1000 * time.Duration((60*1000)/frequency), appId: appId, rm: rm}
	ur, _ := url.Parse(host)
	room.schema = ur.Scheme
	room.address = ur.Host
	// set initial ping interval
	room.pingInterval = 25000
	room.httpTimeout = httpTimeout
	room.wsTimeout = wsTimeout
	room.expireTime = 1440
	room.sdkVersion = rm.sdkVersion
	room.condMutex = &sync.Mutex{}
	room.cond = sync.NewCond(room.condMutex)
	room.msgPool = &sync.Pool{New: func() interface{} {
		pd := new(poolData)
		pd.buf = make([]byte, 4096)
		return pd
	}}

	for i := 0; i < room.usersCap; i++ {
		u := newUser(room)
		if i == 0 {
			u.hostCoWatch = true
		}
		room.users = append(room.users, u)
	}
	return room
}

// requestCreateRoom is that the roomUnit try to create a room on the server
func (p *roomUnit) requestCreateRoom(ctx context.Context) error {
	strings.TrimSuffix(p.address, "/")
	uri := p.schema + "://" + p.address + "/" + "createRoom"

	// p.preRequest()
	start := time.Now()

	roomId := getHostId()
	// construct post body json
	bd := make(map[string]string)
	for k, v := range rm.createRoomExtraData {
		bd[k] = v
	}
	bd["hostUid"] = fmt.Sprintf("%d", getHostId())
	bd["appId"] = p.appId
	bd["version"] = p.sdkVersion
	s, e := json.Marshal(bd)
	if e != nil {
		return e
	}
	log.Debugln("create room, post body is ", string(s))
	req, _ := http.NewRequest("POST", uri, bytes.NewBuffer(s))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("content-length", fmt.Sprintf("%d", len(s)))
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("accept", "*/*")

	req.Header.Set("referer", "https://visualon.com/")
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-site")
	req.Header.Set("sec-ch-ua", "\" Not A;Brand\";v=\"99\", \"Chromium\";v=\"90\", \"Microsoft Edge\";v=\"90\"")
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("dnt", "1")
	req.Close = true // https://stackoverflow.com/a/23963271/16747223

	for k, v := range rm.httpHeaders {
		req.Header.Set(k, v)
	}
	log.Debugln("http header :", req.Header)

	tr := getTransport()
	defer putTransport(tr) // put transport
	newClient := func() *http.Client {
		if p.schema == "https" {
			return &http.Client{Transport: tr, Timeout: p.httpTimeout}
		} else {
			return &http.Client{Timeout: p.httpTimeout}
		}
	}()

	resp, err := newClient.Do(req)
	if err != nil {
		log.Errorln("Failed to post, err = ", err)
		return err
	}
	p.connectionDuration = time.Since(start)
	roomRaw, err := func() ([]byte, error) {

		defer newClient.CloseIdleConnections()
		defer resp.Body.Close()
		return ioutil.ReadAll(resp.Body)
	}()

	if err != nil {
		log.Errorln(err)
		return err
	}

	log.Debugln("roomRaw: ", string(roomRaw))
	// unmarshal
	room := new(roomInfo)
	err = json.Unmarshal(roomRaw, room)
	if err != nil {
		log.Errorln("create room: parsed response failed, response data:", string(roomRaw))
		return err
	} else if room.Name == "" {
		log.Errorln("create room: response is not wanted")
		return err
	}
	p.ns = room.Name
	p.roomId = roomId
	logF.WithFields(logrus.Fields{
		"namespace": room.Name,
		"roomId":    roomId,
	}).Debugln("created room ok")
	// Note: 房间创建完成后，即产生第一个 userInfo， 也是 Address
	//userHost := newUser(p)
	//userHost.hostCoWatch = true
	//userHost.uid = roomId
	//userHost.expireTimer = time.NewTicker(24 * time.Hour)
	//p.users = append(p.users, userHost)

	// add room to rm
	p.rm.lockRoom.Lock()
	p.rm.Rooms = append(p.rm.Rooms, p)
	p.rm.lockRoom.Unlock()
	// go joinRoom(ctx, p, p.users[0], false, nil, nil)
	return nil
}

// preRequest is used for fetch method.
// for some version, it maybe has options method to requestCreateRoom
func (p *roomUnit) preRequest() {
	strings.TrimSuffix(p.address, "/")
	uri := p.schema + "://" + p.address + "/" + "createRoom"

	// request options method
	preReq, _ := http.NewRequest("OPTIONS", uri, nil)
	preReq.Header.Set("access-control-request-headers", "content-type")
	preReq.Header.Set("access-control-request-method", "POST")
	preReq.Header.Set("accept", "*/*")
	preReq.Header.Set("userInfo-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.93 Safari/537.36 Edg/90.0.818.56")
	preReq.Header.Set("Accept-Encoding", "gzip, deflate, br")
	preReq.Header.Set("origin", "https://cowatch.visualon.cn:8080")
	preReq.Header.Set("referer", "https://cowatch.visualon.cn:8080/")
	preReq.Header.Set("sec-fetch-dest", "empty")
	preReq.Header.Set("sec-fetch-mode", "cors")
	preReq.Header.Set("sec-fetch-site", "same-site")

	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	newClient := func() *http.Client {
		if p.schema == "https" {
			return &http.Client{Transport: tr, Timeout: p.httpTimeout}
		} else {
			return &http.Client{Timeout: p.httpTimeout}
		}

	}()
	_, err := newClient.Do(preReq)
	if err != nil {
		log.Errorln("Failed to send OPTIONS method, err = ", err)
	}
	// put transport
	putTransport(tr)
	// 如果 err != nil 则不能 close body，此处可以省略
}

// doesn't deploy it now
func usersOnlineGuard() {

}
