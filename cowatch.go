package main

import (
	"fmt"
	"os"
	"path/filepath"

	color "github.com/fatih/color"
	cb "github.com/garden4hu/cowatchbenchmark"
)

func run(conf *Config) error {
	if 0 == conf.Room {
		return nil
	}
	rm := cb.NewRoomManager(conf.Host, conf.Room, conf.User, conf.Len, conf.Freq)
	rm.RequestRoomsFromServer(conf.StartTimeRoom)
	result := fmt.Sprintf("\n[INFO] Create room: %d, %d%% of target room:(%d), time consuming per room:%s\n", len(rm.Rooms), 100*len(rm.Rooms)/conf.Room, conf.Room, rm.GetCreatingRoomAvgDuration().String())
	color.Set(color.FgYellow)
	_, _ = fmt.Fprintf(os.Stdout, result)
	color.Unset()

	resultFile := filepath.Join(filepath.Dir(os.Args[0]), "result.txt")
	// write the result to local directory
	f, err := os.Create(resultFile)
	if err == nil {
		defer f.Close()
		_, _ = f.Write([]byte(result))
	}
	return nil
}
