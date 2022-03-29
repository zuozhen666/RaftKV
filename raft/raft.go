package raft

type Raft struct {
	id int // raft node id
	// persistent state on all servers
	// updated on stable storage before responding to RPCs
	currentTerm int   // latest term server has seen(initialized to 0 on first boot, increases monotonically)
	votedFor    int   // candidateId that received vote in current term(or null if none)
	logs        []log // log entries; each entry contains command for state machine, and term when entry was received by leader(first index is 1)
	// volatile state on all servers
	commitIndex int // index of highest log entry known to be committed(initialized to 0, increases monotonically)
	lastApplied int // index of highest log entry applied to state machine(initialized to 0, increases monotonically)
	// volatile state on leaders
	// reinitialized after election
	nextIndex  []int // for each server, index of the next log entry to send to that server(initialized to leader last log index + 1)
	matchIndex []int // for each server, index of highest log entry known to be replicated on server(intialized to 0, increases monotonically)
}
