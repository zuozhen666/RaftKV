package raft

import (
	"log"
	"math/rand"
	"time"
)

type State uint

const (
	Follower  State = 1
	Candidate State = 2
	Leader    State = 3
)

type Entry struct {
	Index int    `json:"index"`
	Term  int    `json:"term"`
	Key   string `json:"key"`
	Value string `json:"val"`
}

type RequestVoteArgs struct {
	Term         int    `json:"term"`
	CandidateID  string `json:"candidateId"`
	LastLogIndex int    `json:"lastLogIndex"`
	LastLogTerm  int    `json:"lastLogTerm"`
}

type RequestVoteRes struct {
	Term        int  `json:"term"`
	VoteGranted bool `json:"voteGranted"`
}

type AppendEntriesArgs struct {
	Term         int     `json:"term"`
	LeaderID     string  `json:"leaderId"`
	PrevLogIndex int     `json:"prevLogIndex"`
	PrevLogTerm  int     `json:"prevLogTerm"`
	Entries      []Entry `json:"entries"`
	LeaderCommit int     `json:"leaderCommit"`
}

type AppendEntriesRes struct {
	Term    int  `json:"term"`
	Success bool `json:"success"`
}

type Raft struct {
	ID          string
	CurrentTerm int
	State       State
	VotedFor    string
	Entry       []Entry
	CommitIndex int
	LastApplied int
	NextIndex   map[string]int
	MathchIndex map[string]int
	Peers       []string

	requestVoteFunc   func(peer string, request RequestVoteArgs) (RequestVoteRes, error)
	appendEntriesFunc func(peer string, request AppendEntriesArgs) (AppendEntriesRes, error)

	restartElectionTicker chan int
}

func NewRaft(id string, peers []string,
	requestVoteFunc func(peer string, request RequestVoteArgs) (RequestVoteRes, error),
	appendEntriesFunc func(peer string, request AppendEntriesArgs) (AppendEntriesRes, error),
) *Raft {
	r := Raft{
		ID:                    id,
		Peers:                 peers,
		State:                 Follower,
		NextIndex:             make(map[string]int),
		MathchIndex:           make(map[string]int),
		requestVoteFunc:       requestVoteFunc,
		appendEntriesFunc:     appendEntriesFunc,
		restartElectionTicker: make(chan int),
	}
	for _, peer := range r.Peers {
		r.NextIndex[peer] = 1
		r.MathchIndex[peer] = 0
	}
	r.start()
	return &r
}

func (r *Raft) start() {
	r.startElectionTicker()
	r.startHeartbeatTicker()
}

func (r *Raft) startElectionTicker() {
	electionTicker := time.NewTicker(time.Duration(rand.Int()%1000+4000) * time.Millisecond)
	go func() {
		for {
			select {
			case <-electionTicker.C:
				r.handleElectionTimeout()
			case <-r.restartElectionTicker:
				go r.startElectionTicker()
				return
			}
		}
	}()
}

func (r *Raft) startHeartbeatTicker() {
	heartbeatTicker := time.NewTicker(3 * time.Second)
	go func() {
		for {
			select {
			case <-heartbeatTicker.C:
				r.sendHeartbeats()
			}
		}
	}()
}

func (r *Raft) handleElectionTimeout() {
	if r.State == Leader {
		return
	}
	log.Printf("Node: %s HandleElectionTimeout\n", r.ID)
	r.convertToCandidate()
	r.startElection()
}

func (r *Raft) startElection() {
	log.Printf("Node %s start election", r.ID)
	r.CurrentTerm++
	r.VotedFor = r.ID
	var votesGranted = 1
	for _, peer := range r.Peers {
		requestVoteArgs := RequestVoteArgs{
			Term:         r.CurrentTerm,
			CandidateID:  r.ID,
			LastLogIndex: r.getLastIndex(),
			LastLogTerm:  r.getLastTerm(),
		}
		requestVoteRes, err := r.requestVoteFunc(peer, requestVoteArgs)
		if err != nil {
			log.Printf("Request vote to peer %v err: %v\n", peer, err)
		} else {
			r.updateTermIfNeed(requestVoteRes.Term)
			if r.State != Candidate {
				return
			}
			if requestVoteRes.VoteGranted {
				votesGranted++
			}
		}
	}
	peers := len(r.Peers) + 1
	if votesGranted > peers/2 {
		log.Printf("Node %v granted majority of votes %v of %v", r.ID, votesGranted, peers)
		r.convertToLeader()
	} else {
		log.Printf("Node %v loose election, get votes %v of %v", r.ID, votesGranted, peers)
	}
}

func (r *Raft) sendHeartbeats() {
	if r.State != Leader {
		return
	}
	for _, peer := range r.Peers {
		go func(peer string) {
			var appendEntriesArgs = AppendEntriesArgs{
				Term:     r.CurrentTerm,
				LeaderID: r.ID,
			}
			var appendEntriesRes AppendEntriesRes
			if r.MathchIndex[peer] == r.getLastIndex() {
				// just heartbeat, not append entries
				appendEntriesRes, _ = r.appendEntriesFunc(peer, appendEntriesArgs)
				r.updateTermIfNeed(appendEntriesRes.Term)
			} else {
				r.buildAppendEntriesArgs(&appendEntriesArgs, r.NextIndex[peer])
				appendEntriesRes, _ = r.appendEntriesFunc(peer, appendEntriesArgs)
				for !appendEntriesRes.Success {
					r.NextIndex[peer]--
					r.buildAppendEntriesArgs(&appendEntriesArgs, r.NextIndex[peer])
					appendEntriesRes, _ = r.appendEntriesFunc(peer, appendEntriesArgs)
				}
				r.MathchIndex[peer] = r.NextIndex[peer] - 1
			}
		}(peer)
	}
}

func (r *Raft) buildAppendEntriesArgs(appendEntriesArgs *AppendEntriesArgs, nextIndex int) {
	appendEntriesArgs.Entries = r.Entry[nextIndex:]
	appendEntriesArgs.PrevLogIndex = r.Entry[nextIndex-1].Index
	appendEntriesArgs.PrevLogTerm = r.Entry[nextIndex-1].Term
}

func (r *Raft) HandleRequestVote(requestVoteArgs RequestVoteArgs) RequestVoteRes {
	r.updateTermIfNeed(requestVoteArgs.Term)
	var requestVoteRes = RequestVoteRes{
		Term: r.CurrentTerm,
	}
	if r.VotedFor == "" && r.CurrentTerm < requestVoteArgs.Term && r.getLastIndex() <= requestVoteArgs.LastLogIndex && r.getLastTerm() <= requestVoteArgs.LastLogTerm {
		r.VotedFor = requestVoteArgs.CandidateID
		requestVoteRes.VoteGranted = true
		r.resetElectionTimer()
		return requestVoteRes
	}
	requestVoteRes.VoteGranted = false
	return requestVoteRes
}

func (r *Raft) HandleAppendEntries(appendEntriesArgs AppendEntriesArgs) AppendEntriesRes {
	r.updateTermIfNeed(appendEntriesArgs.Term)
	r.resetElectionTimer()
	// TODO: log replication
	var appendEntriesRes = AppendEntriesRes{
		Term: r.CurrentTerm,
	}
	if r.CurrentTerm > appendEntriesArgs.Term {
		appendEntriesRes.Success = false
	} else {
		appendEntriesRes.Success = true
	}
	return appendEntriesRes
}

func (r *Raft) updateTermIfNeed(term int) {
	if r.CurrentTerm < term {
		log.Printf("Node %v update term %v to %v", r.ID, r.CurrentTerm, term)
		r.CurrentTerm = term
		r.convertToFollower()
	}
}

func (r *Raft) resetElectionTimer() {
	log.Printf("Node %s resetElectionTimer", r.ID)
	r.restartElectionTicker <- 1
}

func (r *Raft) convertToCandidate() {
	log.Printf("Node %s converting to Candidate", r.ID)
	r.State = Candidate
	r.resetElectionTimer()
}
func (r *Raft) convertToFollower() {
	log.Printf("Node %s converting to Follower", r.ID)
	r.State = Follower
	r.VotedFor = ""
	r.resetElectionTimer()
}
func (r *Raft) convertToLeader() {
	log.Printf("Node %s converting to Leader", r.ID)
	r.State = Leader
	for _, peer := range r.Peers {
		r.NextIndex[peer] = r.getLastIndex() + 1
		r.MathchIndex[peer] = 0
	}
	r.sendHeartbeats()
}

func (r *Raft) getLastIndex() int {
	l := len(r.Entry)
	if l == 0 {
		return -1
	}
	return r.Entry[l-1].Index
}

func (r *Raft) getLastTerm() int {
	l := len(r.Entry)
	if l == 0 {
		return -1
	}
	return r.Entry[l-1].Term
}
