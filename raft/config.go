package raft

type State uint32

const (
	Follower  State = 1
	Candidate State = 2
	Leader    State = 3
)

type RequestVoteArgs struct {
	term         int // candidate's term
	candidateId  int // candidate requesting vote
	lastLogIndex int // index of candidate' last log entry
	lastLogTerm  int // term of candidate received vote
}

type RequestVoteRes struct {
	term        int  // currentTerm, for candidate to update itself
	voteGranted bool // true means candidate received vote
}

type entry struct {
	// TODO:
}

type log struct {
	// TODO:
}

type AppendEntriesArgs struct {
	term         int     // leader's term
	leaderId     int     // so follower can redirect clients
	prevLogIndex int     // index of log entry immediately preceding new ones
	prevLogTerm  int     // term of prevLogIndex entry
	entries      []entry // log entries to store (empty for heartbeat;)
	leaderCommit int     // leader's commitIndex
}

type AppendEntriesRes struct {
	term    int  // currentTerm, for leader to update itself
	success bool // true if follower contained entry matching precLogIndex and prevLogTerm
}
