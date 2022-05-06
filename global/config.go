package global

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

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
	Mutex        sync.RWMutex
	LiveNum      int
	LeaderID     string
	LeaderKvPort string
	GlobalC      chan<- bool
	OtherPeers   map[string]struct{}
	MaxPeers     map[string]struct{}
}

var ClusterMeta cluster
var Node node
var client = http.Client{}

type test struct {
	Term int `json:"term"`
}

func DaemProcess() {
	ticker := time.NewTicker(6 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				livePeers := make(map[string]struct{})
				reqJson, _ := json.Marshal(test{
					Term: -1,
				})
				for peer, _ := range ClusterMeta.MaxPeers {
					if _, err := client.Post("http://"+peer+"/raft/request-vote", "application/json", bytes.NewBuffer(reqJson)); err == nil {
						livePeers[peer] = struct{}{}
					}
				}
				if len(livePeers)+1 != ClusterMeta.LiveNum {
					log.Printf("[global]update live peers %v to %v", ClusterMeta.OtherPeers, livePeers)
					ClusterMeta.Mutex.Lock()
					ClusterMeta.OtherPeers = livePeers
					ClusterMeta.LiveNum = len(livePeers) + 1
					ClusterMeta.Mutex.Unlock()
					ClusterMeta.GlobalC <- true
				}
			}
		}
	}()
}
