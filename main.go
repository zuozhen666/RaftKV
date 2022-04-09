package main

import (
	"RaftKV/raft"
	"os"
)

func main() {
	// httpServer := &http.Server{
	// 	Addr: ":8080",
	// 	Handler: &kvServer{
	// 		store: NewKvStore(),
	// 	},
	// }
	// httpServer.ListenAndServe()
	args := os.Args[1:]
	if len(args) < 1 {
		panic("args not fail")
	}
	client := raft.NewRaftClient()
	r := raft.NewRaft(args[0], args[1:], client.RequestVote, client.AppendEntries)
	server := raft.NewRaftServer(r)
	server.Start(args[0])
}
