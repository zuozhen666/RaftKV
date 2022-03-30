package global

type nodeInfo struct {
	Id   int
	Port string
}

type globalInfo struct {
	Nodes []nodeInfo
}

// 全局信息
var GlobalInfo globalInfo

func InitGlobalInfo() {}

func GetLeaderPort() string { return "" }
