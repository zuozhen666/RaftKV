package raft

// State
type State int

const (
	Follower State = iota + 1
	Candidate
	Leader
)

// LogEntry
type LogEntry struct {
	LogTerm  int
	LogIndex int
	LogCMD   interface{}
}

// Vote
type VoteArgs struct {
	Term        int
	CandidateID int
}

type VoteReply struct {
	Term        int
	VoteGranted bool
}

// Heartbeat
type HeartbeatArgs struct {
	Term         int
	LeaderID     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	LeaderCommit int
}

type HeartbeatReply struct {
	Success   bool
	Term      int
	NextIndex int
}
