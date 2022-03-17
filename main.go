package main

import (
	"RaftKV/raft"

	"flag"
	"strings"
)

func main() {
	port := flag.String("port", ":9091", "rpc listen port")
	cluster := flag.String("cluster", "127.0.0.1:9091", "comma sep")
	id := flag.Int("id", 1, "node ID")

	flag.Parse()
	clusters := strings.Split(*cluster, ",")

	ns := make(map[int]*raft.Node)
	for k, v := range clusters {
		ns[k] = raft.NewNode(v)
	}

	r := raft.NewRaft(*id, ns)
	r.Rpc(*port)
	r.Start()

	select {}
}
