package config

type Kv struct {
	Key string
	Val string
	Op  string
}

type cluster struct {
	LiveNum  int
	LeaderID string
}

var ClusterMeta cluster
