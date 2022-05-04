package main

import (
	"RaftKV/config"
	"RaftKV/kvserver"
	"RaftKV/raft"
	"net/http"
	"os"
)

func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		panic("args not fail")
	}
	proposeC := make(chan config.Kv)
	commitC := make(chan config.Kv)
	httpServer := &http.Server{
		Addr: args[0],
		Handler: &kvserver.KvServer{
			Store: kvserver.NewKvStore(proposeC, commitC),
		},
	}
	httpServer.ListenAndServe()
	client := raft.NewRaftClient()
	r := raft.NewRaft(args[1], args[2:], proposeC, commitC, client.RequestVote, client.AppendEntries)
	server := raft.NewRaftServer(r)
	server.Start(args[1])
}
