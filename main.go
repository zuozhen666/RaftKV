package main

import (
	"RaftKV/raft"
	"net/http"
	"os"
)

func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		panic("args not fail")
	}
	httpServer := &http.Server{
		Addr: args[0],
		Handler: &kvServer{
			store: NewKvStore(),
		},
	}
	httpServer.ListenAndServe()
	client := raft.NewRaftClient()
	r := raft.NewRaft(args[1], args[2:], client.RequestVote, client.AppendEntries)
	server := raft.NewRaftServer(r)
	server.Start(args[1])
}
