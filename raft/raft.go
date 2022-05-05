package raft

import (
	"RaftKV/global"
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
	Op    string `json:"op"`
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
	proposeC              <-chan global.Kv
	commitC               chan<- global.Kv
	ptr                   int
}

func NewRaft(id string, peers []string, proposeC <-chan global.Kv, commitC chan<- global.Kv,
	requestVoteFunc func(peer string, request RequestVoteArgs) (RequestVoteRes, error),
	appendEntriesFunc func(peer string, request AppendEntriesArgs) (AppendEntriesRes, error),
) *Raft {
	r := Raft{
		ID:                    id,
		Peers:                 peers,
		State:                 Follower,
		VotedFor:              "",
		Entry:                 make([]Entry, 0),
		CommitIndex:           -1,
		NextIndex:             make(map[string]int),
		MathchIndex:           make(map[string]int),
		requestVoteFunc:       requestVoteFunc,
		appendEntriesFunc:     appendEntriesFunc,
		restartElectionTicker: make(chan int, 1),
		proposeC:              proposeC,
		commitC:               commitC,
		ptr:                   -1,
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
	go r.readPropose()
	r.commitCycle()
}

func (r *Raft) readPropose() {
	for propose := range r.proposeC {
		log.Printf("[raft module]receive new propose %v", propose)
		r.Entry = append(r.Entry, Entry{
			Index: r.ptr + 1,
			Term:  r.CurrentTerm,
			Key:   propose.Key,
			Value: propose.Val,
			Op:    propose.Op,
		})
		r.ptr++
	}
}

func (r *Raft) commit(lastCommitIndex int) {
	log.Printf("raft module reach a consensus, commit %v", r.Entry[lastCommitIndex+1:r.CommitIndex])
	for lastCommitIndex < r.CommitIndex {
		r.commitC <- global.Kv{
			Key: r.Entry[lastCommitIndex+1].Key,
			Val: r.Entry[lastCommitIndex+1].Value,
			Op:  r.Entry[lastCommitIndex+1].Op,
		}
		lastCommitIndex++
	}
}

func (r *Raft) commitCycle() {
	commitTicker := time.NewTicker(1 * time.Second)
	go func() {
		for {
			select {
			case <-commitTicker.C:
				if r.State == Leader {
					if global.ClusterMeta.LiveNum == 1 {
						if r.getLastIndex() > r.CommitIndex {
							lastCommitIndex := r.CommitIndex
							r.CommitIndex = r.getLastIndex()
							r.commit(lastCommitIndex)
						}
					} else {
						count := 0
						for _, index := range r.MathchIndex {
							if index >= r.CommitIndex+1 {
								count++
							}
						}
						if count >= global.ClusterMeta.LiveNum/2 {
							r.CommitIndex++
							r.commit(r.CommitIndex - 1)
						}
					}
				}
			}
		}
	}()
}

func (r *Raft) startElectionTicker() {
	randInt := rand.Int()%1000 + 4000
	electionTicker := time.NewTicker(time.Duration(randInt) * time.Millisecond)
	log.Printf("reset electionTimeout: %v Millisecond\n", randInt)
	go func() {
		for {
			select {
			case <-electionTicker.C:
				log.Printf("Node %v election time out\n", r.ID)
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
			log.Printf("CurrentTerm = %v, reveiceRes = %v\n", r.CurrentTerm, requestVoteRes)
			r.updateTermIfNeed(requestVoteRes.Term)
			if r.State != Candidate {
				return
			}
			if requestVoteRes.VoteGranted {
				votesGranted++
			}
		}
	}
	if votesGranted > global.ClusterMeta.LiveNum/2 {
		log.Printf("Node %v granted majority of votes %v of %v", r.ID, votesGranted, global.ClusterMeta.LiveNum)
		r.convertToLeader()
	} else {
		log.Printf("Node %v loose election, get votes %v of %v", r.ID, votesGranted, global.ClusterMeta.LiveNum)
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
			lastIndex := r.getLastIndex()
			if r.MathchIndex[peer] == lastIndex || lastIndex == -1 {
				appendEntriesRes, _ = r.appendEntriesFunc(peer, appendEntriesArgs)
				log.Printf("Node %v appendEntries to Node %v\n", r.ID, peer)
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
	appendEntriesArgs.PrevLogIndex = r.Entry[nextIndex-2].Index
	appendEntriesArgs.PrevLogTerm = r.Entry[nextIndex-2].Term
	appendEntriesArgs.Entries = r.Entry[:appendEntriesArgs.PrevLogIndex]
}

func (r *Raft) HandleRequestVote(requestVoteArgs RequestVoteArgs) RequestVoteRes {
	r.updateTermIfNeed(requestVoteArgs.Term)
	var requestVoteRes = RequestVoteRes{
		Term: r.CurrentTerm,
	}
	if r.VotedFor == "" && r.CurrentTerm <= requestVoteArgs.Term && r.getLastIndex() <= requestVoteArgs.LastLogIndex && r.getLastTerm() <= requestVoteArgs.LastLogTerm {
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
	log.Printf("Node: %v receive appendEntriesMsg from Node: %v\n", r.ID, appendEntriesArgs.LeaderID)
	r.resetElectionTimer()
	var appendEntriesRes = AppendEntriesRes{
		Term: r.CurrentTerm,
	}
	if appendEntriesArgs.Entries == nil {
		appendEntriesRes.Success = r.CurrentTerm <= appendEntriesArgs.Term
		return appendEntriesRes
	}
	if !r.isMatch(appendEntriesArgs.PrevLogIndex, appendEntriesArgs.PrevLogTerm) {
		appendEntriesRes.Success = false
		return appendEntriesRes
	}
	r.Entry = append(r.Entry[:appendEntriesArgs.PrevLogIndex], appendEntriesArgs.Entries...)
	r.updateCommitIndexIfNeed(appendEntriesArgs.LeaderCommit)
	return appendEntriesRes
}

func (r *Raft) updateCommitIndexIfNeed(leaderCommit int) {
	if r.getLastIndex() >= leaderCommit {
		lastCommitIndex := r.CommitIndex
		r.CommitIndex = leaderCommit
		r.commit(lastCommitIndex)
	}
}

func (r *Raft) isMatch(prevLogIndex, prevLogTerm int) bool {
	for _, entry := range r.Entry {
		if prevLogIndex == entry.Index && prevLogTerm == entry.Term {
			return true
		}
	}
	return false
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
