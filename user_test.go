package main

import (
	_ "net/http/pprof"
	"testing"
)

func TestRoomUnit_Chat(t *testing.T) {
	// create r witch server, https
	//r := newRoom("http://cowatch_server", 25*time.Second, 45*time.Second, 300, 20, 1, "appid", 1)
	//if err := r.request(); err != nil {
	//	t.Error("failed to finish request")
	//}
	//log.SetFlags(0)
	//log.SetOutput(ioutil.Discard)
	//fmt.Println("room roomName = ", r.roomName)
	//ctx, cancel := context.WithCancel(context.Background())
	//defer cancel()
	//ch := make(chan struct{})
	//
	//go r.usersConnection(ch, true,ctx)
	//close(ch)
	//// pprof
	//go func() {
	//	fmt.Println("pprof start...")
	//	fmt.Println(http.ListenAndServe("127.0.0.1:9876", nil))
	//}()
}
