package main

import (
	"RaftKV/global"
	"RaftKV/kvserver"
	"RaftKV/raft"
	"log"
	"net/http"
	"os"
)

func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		panic("args not fail")
	}
	proposeC := make(chan global.Kv)
	commitC := make(chan global.Kv)
	// kv server
	httpServer := &http.Server{
		Addr: args[0],
		Handler: &kvserver.KvServer{
			Store: kvserver.NewKvStore(proposeC, commitC),
		},
	}
	go httpServer.ListenAndServe()
	// raft node
	global.ClusterMeta.LiveNum = len(args[1:])
	log.Printf("Cluster Live node: %v", args[1:])
	client := raft.NewRaftClient()
	r := raft.NewRaft(args[1], args[2:], proposeC, commitC, client.RequestVote, client.AppendEntries)
	server := raft.NewRaftServer(r)
	server.Start(args[1])
}
