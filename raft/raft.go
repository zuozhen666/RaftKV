package raft

import (
	"RaftKV/global"
)

type Raft struct {
	id    int // raft node id
	state State
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

	election chan bool
	App      chan bool
}

// rpc method: 处理RequestVote
func (r *Raft) RequestVote(request RequestVoteArgs, reply *RequestVoteRes) error {
	if request.term < r.currentTerm {
		reply.term = r.currentTerm
		reply.voteGranted = false
		return nil
	}
	if r.votedFor != -1 {
		reply.term = r.currentTerm
		reply.voteGranted = false
		return nil
	}
	// A的节点日志至少比B新：满足A.lastTerm >= B.lastTerm && A.lastIndex >= B.lastIndex
	if request.lastLogTerm >= r.getLastTerm() && request.lastLogIndex >= r.getLastIndex() {
		reply.term = r.currentTerm
		reply.voteGranted = true
		r.votedFor = request.candidateId
		r.currentTerm = request.term
	}
	return nil
}

// rpc method: 处理AppendEntries
func (r *Raft) AppendEntries(request AppendEntriesArgs, reply *AppendEntriesRes) error {
	return nil
}

func (r *Raft) getLastIndex() int {
	if len := len(r.logs); len != 0 {
		return r.logs[len-1].Index
	}
	return -1
}
func (r *Raft) getLastTerm() int {
	if len := len(r.logs); len != 0 {
		return r.logs[len-1].Term
	}
	return -1
}

func (r *Raft) sendRequestVote(serverPort string) {

}

func (r *Raft) sendHeartBeat(serverPort string) {

}

func (r *Raft) broadcastRequestVote() {
	for _, nodeInfo := range global.GlobalInfo.Nodes {
		if nodeInfo.Id != r.id {
			go func(port string) {
				r.sendRequestVote(port)
			}(nodeInfo.Port)
		}
	}
}

func (r *Raft) broadcastHeartbeat() {
	for _, nodeInfo := range global.GlobalInfo.Nodes {
		if nodeInfo.Id != r.id {
			go func(port string) {
				r.sendHeartBeat(port)
			}(nodeInfo.Port)
		}
	}
}
