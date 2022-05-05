package global

import "time"

type Kv struct {
	Key string
	Val string
	Op  string
}

type node struct {
	KvPort      string
	RaftAddress string
}

type cluster struct {
	LiveNum      int
	LeaderID     string
	LeaderKvPort string
	OtherPeers   []string
}

var ClusterMeta cluster
var Node node

func DaemProcess() {
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:

			}
		}
	}()
}
