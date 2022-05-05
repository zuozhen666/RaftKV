package main

import (
	"RaftKV/global"
	"RaftKV/kvserver"
	"RaftKV/raft"
	"net/http"
	"os"
)

func main() {
	args := os.Args[1:]
	if len(args) < 2 {
		panic("args not fail")
	}
	proposeC := make(chan global.Kv)
	commitC := make(chan global.Kv)
	global.ClusterMeta.LiveNum = len(args[1:])
	global.ClusterMeta.OtherPeers = args[2:]
	global.Node.KvPort = args[0]
	global.Node.RaftAddress = args[1]
	// kv server
	httpServer := &http.Server{
		Addr: args[0],
		Handler: &kvserver.KvServer{
			Store: kvserver.NewKvStore(proposeC, commitC),
		},
	}
	go httpServer.ListenAndServe()
	// raft node
	client := raft.NewRaftClient()
	r := raft.NewRaft(args[1], proposeC, commitC, client.RequestVote, client.AppendEntries)
	server := raft.NewRaftServer(r)
	server.Start(args[1])
}
