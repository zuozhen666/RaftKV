package raft

import (
	"RaftKV/global"
	"log"
	"math/rand"
	"sync"
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
	LeaderKvPort string  `json:"leaderkvport"`
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
	MatchIndex  map[string]int

	requestVoteFunc   func(peer string, request RequestVoteArgs) (RequestVoteRes, error)
	appendEntriesFunc func(peer string, request AppendEntriesArgs) (AppendEntriesRes, error)

	restartElectionTicker chan int
	proposeC              <-chan global.Kv
	commitC               chan<- global.Commit
	globalC               <-chan bool
	ptr                   int
}

func NewRaft(id string, proposeC <-chan global.Kv, commitC chan<- global.Commit, globalC <-chan bool,
	requestVoteFunc func(peer string, request RequestVoteArgs) (RequestVoteRes, error),
	appendEntriesFunc func(peer string, request AppendEntriesArgs) (AppendEntriesRes, error),
) *Raft {
	r := Raft{
		ID:                    id,
		State:                 Follower,
		VotedFor:              "",
		Entry:                 make([]Entry, 0),
		CommitIndex:           -1,
		NextIndex:             make(map[string]int),
		MatchIndex:            make(map[string]int),
		requestVoteFunc:       requestVoteFunc,
		appendEntriesFunc:     appendEntriesFunc,
		restartElectionTicker: make(chan int, 1),
		proposeC:              proposeC,
		commitC:               commitC,
		globalC:               globalC,
		ptr:                   -1,
	}
	global.ClusterMeta.Mutex.RLock()
	for peer, _ := range global.ClusterMeta.OtherPeers {
		r.NextIndex[peer] = 0
		r.MatchIndex[peer] = -1
	}
	global.ClusterMeta.Mutex.RUnlock()
	r.start()
	return &r
}

func (r *Raft) start() {
	r.startElectionTicker()
	r.startHeartbeatTicker()
	go r.readPropose()
	r.commitCycle()
	r.listenGlobal()
}

func (r *Raft) readPropose() {
	for propose := range r.proposeC {
		log.Printf("[raft module]receive propose request %v from kvserver", propose)
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
	log.Printf("[raft module]reach a consensus, commit %v", r.Entry[lastCommitIndex+1:r.CommitIndex+1])
	for lastCommitIndex < r.CommitIndex {
		commit := global.Commit{
			Kv: global.Kv{
				Key: r.Entry[lastCommitIndex+1].Key,
				Val: r.Entry[lastCommitIndex+1].Value,
				Op:  r.Entry[lastCommitIndex+1].Op,
			},
			WG: &sync.WaitGroup{},
		}
		commit.WG.Add(1)
		r.commitC <- commit
		commit.WG.Wait()
		log.Printf("[raft module]apply entry %v to state machine", r.Entry[lastCommitIndex+1])
		r.LastApplied = lastCommitIndex + 1
		lastCommitIndex++
	}
}

func (r *Raft) commitCycle() {
	commitTicker := time.NewTicker(5 * time.Second)
	go func() {
		for {
			select {
			case <-commitTicker.C:
				if r.State == Leader {
					global.ClusterMeta.Mutex.RLock()
					if global.ClusterMeta.LiveNum == 1 {
						if r.getLastIndex() > r.CommitIndex {
							lastCommitIndex := r.CommitIndex
							r.CommitIndex = r.getLastIndex()
							log.Printf("[raft module]leader update commit index %d to %d", lastCommitIndex, r.CommitIndex)
							r.commit(lastCommitIndex)
						}
					} else {
						count := 0
						for _, index := range r.MatchIndex {
							if index >= r.CommitIndex+1 {
								count++
							}
						}
						if count >= global.ClusterMeta.LiveNum/2 {
							log.Printf("[raft module]leader update commit index %d to %d", r.CommitIndex, r.CommitIndex+1)
							r.CommitIndex++
							r.commit(r.CommitIndex - 1)
						}
					}
					global.ClusterMeta.Mutex.RUnlock()
				}
			}
		}
	}()
}

func (r *Raft) listenGlobal() {
	go func() {
		for {
			select {
			case <-r.globalC:
				if r.State == Leader {
					global.ClusterMeta.Mutex.RLock()
					log.Println("[raft module]receive global cluster change, update NextIndex and MatchIndex")
					for peer, _ := range r.NextIndex {
						if _, ok := global.ClusterMeta.OtherPeers[peer]; !ok {
							delete(r.NextIndex, peer)
							delete(r.MatchIndex, peer)
						}
					}
					for peer, _ := range global.ClusterMeta.OtherPeers {
						if _, ok := r.NextIndex[peer]; !ok {
							r.NextIndex[peer] = r.getLastIndex() + 1
							r.MatchIndex[peer] = -1
						}
					}
					global.ClusterMeta.Mutex.RUnlock()
				}
			}
		}
	}()
}

func (r *Raft) startElectionTicker() {
	randInt := rand.Int()%1000 + 4000
	electionTicker := time.NewTicker(time.Duration(randInt) * time.Millisecond)
	log.Printf("[raft module]reset electionTimeout: %v Millisecond", randInt)
	go func() {
		for {
			select {
			case <-electionTicker.C:
				if r.State != Leader {
					log.Printf("[raft module]Node %v election time out", r.ID)
					r.handleElectionTimeout()
				}
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
	log.Printf("[raft module]Node: %s HandleElectionTimeout", r.ID)
	r.convertToCandidate()
	r.startElection()
}

func (r *Raft) startElection() {
	log.Printf("[raft module]Node %s start election", r.ID)
	r.CurrentTerm++
	r.VotedFor = r.ID
	var votesGranted = 1
	global.ClusterMeta.Mutex.RLock()
	for peer, _ := range global.ClusterMeta.OtherPeers {
		requestVoteArgs := RequestVoteArgs{
			Term:         r.CurrentTerm,
			CandidateID:  r.ID,
			LastLogIndex: r.getLastIndex(),
			LastLogTerm:  r.getLastTerm(),
		}
		requestVoteRes, err := r.requestVoteFunc(peer, requestVoteArgs)
		if err != nil {
			log.Printf("[raft module]Request vote to peer %v err: %v", peer, err)
		} else {
			log.Printf("[raft module]CurrentTerm = %v, reveiceRes = %v", r.CurrentTerm, requestVoteRes)
			r.updateTermIfNeed(requestVoteRes.Term)
			if r.State != Candidate {
				return
			}
			if requestVoteRes.VoteGranted {
				votesGranted++
			}
		}
	}
	global.ClusterMeta.Mutex.RUnlock()
	if votesGranted > global.ClusterMeta.LiveNum/2 {
		log.Printf("[raft module]Node %v granted majority of votes %v of %v", r.ID, votesGranted, global.ClusterMeta.LiveNum)
		r.convertToLeader()
	} else {
		log.Printf("[raft module]Node %v loose election, get votes %v of %v", r.ID, votesGranted, global.ClusterMeta.LiveNum)
	}
}

func (r *Raft) sendHeartbeats() {
	if r.State != Leader {
		return
	}
	global.ClusterMeta.Mutex.RLock()
	for peer, _ := range global.ClusterMeta.OtherPeers {
		go func(peer string) {
			var appendEntriesArgs = AppendEntriesArgs{
				Term:         r.CurrentTerm,
				LeaderID:     r.ID,
				LeaderCommit: r.CommitIndex,
				LeaderKvPort: global.Node.KvPort,
			}
			var appendEntriesRes AppendEntriesRes
			lastIndex := r.getLastIndex()
			if r.MatchIndex[peer] == lastIndex {
				appendEntriesRes, _ = r.appendEntriesFunc(peer, appendEntriesArgs)
				log.Printf("[raft module]Node %v send heartbeat to Node %v", r.ID, peer)
				r.updateTermIfNeed(appendEntriesRes.Term)
			} else {
				r.buildAppendEntriesArgs(&appendEntriesArgs, r.NextIndex[peer])
				appendEntriesRes, _ = r.appendEntriesFunc(peer, appendEntriesArgs)
				log.Printf("[raft module]Node %v appendEntries to Node %v, Entries: %v", r.ID, peer, appendEntriesArgs.Entries)
				for !appendEntriesRes.Success {
					r.NextIndex[peer]--
					r.buildAppendEntriesArgs(&appendEntriesArgs, r.NextIndex[peer])
					appendEntriesRes, _ = r.appendEntriesFunc(peer, appendEntriesArgs)
					log.Printf("[raft module]Node %v appendEntries(retry) to Node %v, Entries: %v", r.ID, peer, appendEntriesArgs.Entries)
				}
				if len(appendEntriesArgs.Entries) == 0 {
					r.NextIndex[peer] = r.getLastIndex() + 1
				} else {
					r.NextIndex[peer] = appendEntriesArgs.Entries[len(appendEntriesArgs.Entries)-1].Index + 1
				}
				r.MatchIndex[peer] = r.NextIndex[peer] - 1
			}
		}(peer)
	}
	global.ClusterMeta.Mutex.RUnlock()
}

func (r *Raft) buildAppendEntriesArgs(appendEntriesArgs *AppendEntriesArgs, nextIndex int) {
	if nextIndex != 0 {
		appendEntriesArgs.PrevLogIndex = r.Entry[nextIndex-1].Index
		appendEntriesArgs.PrevLogTerm = r.Entry[nextIndex-1].Term
	} else {
		appendEntriesArgs.PrevLogIndex = -1
	}
	appendEntriesArgs.Entries = r.Entry[nextIndex:]
}

func (r *Raft) HandleRequestVote(requestVoteArgs RequestVoteArgs) RequestVoteRes {
	// global daem
	if requestVoteArgs.Term == -1 {
		return RequestVoteRes{
			Term: r.CurrentTerm,
		}
	}
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
	log.Printf("[raft module]Node: %v receive appendEntriesMsg from Node: %v", r.ID, appendEntriesArgs.LeaderID)
	r.resetElectionTimer()
	var appendEntriesRes = AppendEntriesRes{
		Term: r.CurrentTerm,
	}
	if appendEntriesArgs.Entries == nil {
		// heartbeat
		if r.CurrentTerm > appendEntriesArgs.Term {
			appendEntriesRes.Success = false
		} else {
			appendEntriesRes.Success = true
			r.updateCommitIndexIfNeed(appendEntriesArgs.LeaderCommit)
			global.ClusterMeta.Mutex.Lock()
			global.ClusterMeta.LeaderID = appendEntriesArgs.LeaderID
			global.ClusterMeta.LeaderKvPort = appendEntriesArgs.LeaderKvPort
			global.ClusterMeta.Mutex.Unlock()
		}
		return appendEntriesRes
	}
	if r.getLastIndex() == -1 && appendEntriesArgs.PrevLogIndex != -1 {
		appendEntriesRes.Success = false
		return appendEntriesRes
	}
	if r.getLastIndex() != -1 && !r.isMatch(appendEntriesArgs.PrevLogIndex, appendEntriesArgs.PrevLogTerm) {
		appendEntriesRes.Success = false
		return appendEntriesRes
	}
	if r.getLastIndex() != -1 {
		r.Entry = append(r.Entry[:appendEntriesArgs.PrevLogIndex+1], appendEntriesArgs.Entries...)
	} else {
		r.Entry = append(r.Entry, appendEntriesArgs.Entries...)
	}
	appendEntriesRes.Success = true
	log.Printf("[raft module]Node %v receive appendEntriesMsg from leader, current Entry %v", r.ID, r.Entry)
	r.updateCommitIndexIfNeed(appendEntriesArgs.LeaderCommit)
	global.ClusterMeta.Mutex.Lock()
	global.ClusterMeta.LeaderID = appendEntriesArgs.LeaderID
	global.ClusterMeta.LeaderKvPort = appendEntriesArgs.LeaderKvPort
	global.ClusterMeta.Mutex.Unlock()
	return appendEntriesRes
}

func (r *Raft) isMatch(prevLogIndex, prevLogTerm int) bool {
	for _, entry := range r.Entry {
		if prevLogIndex == entry.Index && prevLogTerm == entry.Term {
			return true
		}
	}
	return false
}

func (r *Raft) updateCommitIndexIfNeed(leaderCommit int) {
	if r.getLastIndex() != -1 && r.getLastIndex() >= leaderCommit && r.CommitIndex != leaderCommit {
		lastCommitIndex := r.CommitIndex
		r.CommitIndex = leaderCommit
		log.Printf("[raft module]Node %v update commitIndex %d to %d", r.ID, lastCommitIndex, leaderCommit)
		r.commit(lastCommitIndex)
	}
}

func (r *Raft) updateTermIfNeed(term int) {
	if r.CurrentTerm < term {
		log.Printf("[raft module]Node %v update term %v to %v", r.ID, r.CurrentTerm, term)
		r.CurrentTerm = term
		r.convertToFollower()
	}
}

func (r *Raft) resetElectionTimer() {
	log.Printf("[raft module]Node %s resetElectionTimer", r.ID)
	r.restartElectionTicker <- 1
}

func (r *Raft) convertToCandidate() {
	log.Printf("[raft module]Node %s converting to Candidate", r.ID)
	r.State = Candidate
	r.resetClusterLeaderInfo()
	r.resetElectionTimer()
}
func (r *Raft) convertToFollower() {
	log.Printf("[raft module]Node %s converting to Follower", r.ID)
	r.State = Follower
	r.VotedFor = ""
	r.resetClusterLeaderInfo()
	r.resetElectionTimer()
}
func (r *Raft) convertToLeader() {
	log.Printf("[raft module]Node %s converting to Leader", r.ID)
	r.State = Leader
	r.ptr = len(r.Entry) - 1
	global.ClusterMeta.Mutex.Lock()
	global.ClusterMeta.LeaderID = r.ID
	global.ClusterMeta.LeaderKvPort = global.Node.KvPort
	global.ClusterMeta.Mutex.Unlock()
	r.NextIndex = make(map[string]int)
	r.MatchIndex = make(map[string]int)
	for peer, _ := range global.ClusterMeta.OtherPeers {
		r.NextIndex[peer] = r.getLastIndex() + 1
		r.MatchIndex[peer] = -1
	}
	r.sendHeartbeats()
}

func (r *Raft) resetClusterLeaderInfo() {
	global.ClusterMeta.Mutex.Lock()
	global.ClusterMeta.LeaderID = ""
	global.ClusterMeta.LeaderKvPort = ""
	global.ClusterMeta.Mutex.Unlock()
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
