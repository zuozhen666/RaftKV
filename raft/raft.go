package raft

import (
	"log"
	"math/rand"
	"time"
)

type Raft struct {
	state State

	requestVoteFunc              func(peer string, term int, candidateId string) (int, bool, error)
	appendEntriesFunc            func(peer string, term int) (int, bool, error)
	restartElectionTickerChannel chan int
}

type State struct {
	CurrentTerm   int      `json:"currentTerm"`
	LastHeartbeat int64    `json:"lastHeartbeat"`
	Peers         []string `json:"peers"`
	Role          string   `json:"role"`
	NodeId        string   `json:"nodeId"`
	VotedFor      string   `json:"votedFor"`
	VotesGranted  int      `json:"votesGranted"`
}

func NewRaft(
	requestVoteFunc func(peer string, term int, candidateId string) (int, bool, error),
	appendEntriesFunc func(peer string, term int) (int, bool, error),
	nodeId string,
	peers []string,
) *Raft {
	r := Raft{
		state: State{
			NodeId: nodeId,
			Peers:  peers,
			Role:   "follower",
		},
		requestVoteFunc:              requestVoteFunc,
		appendEntriesFunc:            appendEntriesFunc,
		restartElectionTickerChannel: make(chan int, 100),
	}
	r.start()
	return &r
}

func (r *Raft) State() State {
	return r.state
}

func (r *Raft) start() {
	r.startElectionTicker()
	r.startLeaderHeartbeatsTicker()
}

func (r *Raft) startElectionTicker() {
	var randomMillis = rand.Int()%1000 + 4000
	electionTicker := time.NewTicker(time.Duration(randomMillis) * time.Millisecond)
	go func() {
		for {
			select {
			case <-electionTicker.C:
				r.handleElectionTimeout()
			case <-r.restartElectionTickerChannel:
				go r.startElectionTicker()
				return
			}
		}
	}()
}

func (r *Raft) startLeaderHeartbeatsTicker() {
	leaderHeartbeatTicker := time.NewTicker(3 * time.Second)
	go func() {
		for {
			select {
			case <-leaderHeartbeatTicker.C:
				r.sendHeartbeats()
			}
		}
	}()
}

func (r *Raft) handleElectionTimeout() {
	if r.isLeader() {
		return
	}
	log.Println("Handle election timeout")
	r.convertToCandidate()
	r.startElection()
}

func (r *Raft) sendHeartbeats() {
	for _, peer := range r.state.Peers {
		if !r.isLeader() {
			return
		}
		log.Printf("Sending heartbeat to %v", peer)
		term, _, err := r.appendEntriesFunc(peer, r.state.CurrentTerm)
		if err != nil {
			log.Printf("Error on append entries to peer %v: %v", peer, err)
		} else {
			r.updateTermIfNeeded(term)
		}
	}
	r.updateHeartbeat()
}

func (r *Raft) updateHeartbeat() {
	r.state.LastHeartbeat = time.Now().UnixNano() / int64(time.Millisecond)
}

func (r *Raft) startElection() {
	log.Print("Starting election")
	r.state.CurrentTerm++
	r.state.VotedFor = r.state.NodeId
	r.resetElectionTimer()
	var votesGranted = 1 // voted for myself
	for _, peer := range r.state.Peers {
		log.Printf("Sending request vote to peer %v", peer)
		term, voteGranted, err := r.requestVoteFunc(peer, r.state.CurrentTerm, r.state.NodeId)
		log.Printf("Received %v term and vote %v", term, votesGranted)
		if err != nil {
			log.Printf("Error on request vote to peer %v: %v", peer, err)
		} else {
			r.updateTermIfNeeded(term)
			if !r.isCandidate() {
				return
			}
			if voteGranted {
				votesGranted++
			}
		}
	}
	r.state.VotesGranted = votesGranted
	allPeers := len(r.state.Peers) + 1
	if votesGranted > allPeers/2 {
		log.Printf("Granted majority of votes %v of %v", votesGranted, allPeers)
		r.convertToLeader()
	} else {
		log.Printf("Did not granted majority of votes: %v of %v", votesGranted, allPeers)
	}
}

func (r *Raft) convertToLeader() {
	log.Print("Converting to leader")
	r.state.Role = "leader"
	r.sendHeartbeats()
	r.startLeaderHeartbeatsTicker()
}
func (r *Raft) convertToCandidate() {
	log.Print("Converting to candidate")
	r.state.Role = "candidate"
	r.resetElectionTimer()
}

func (r *Raft) convertToFollower() {
	log.Print("Converting to follower")
	r.state.Role = "follower"
	r.state.VotedFor = ""
	r.resetElectionTimer()
}

func (r *Raft) RequestVote(term int, candidateId string) bool {
	log.Printf("Requesting vote for candidate %v and term %v", candidateId, term)
	r.updateTermIfNeeded(term)
	if term < r.state.CurrentTerm {
		log.Printf("Vote rejected. Received stale term %v than current %v.", term, r.state.CurrentTerm)
		return false
	}
	if r.state.VotedFor != "" {
		log.Printf("Vote rejected. Already voted for %v", r.state.VotedFor)
		return false
	}
	r.state.VotedFor = candidateId
	log.Printf("Voted granted")
	r.resetElectionTimer()
	return true
}

func (r *Raft) AppendEntries(term int) bool {
	log.Printf("Appending entry for term %v", term)
	r.updateTermIfNeeded(term)
	r.resetElectionTimer()
	if term < r.state.CurrentTerm {
		log.Printf("Received lower term %v than current %v", term, r.state.CurrentTerm)
		return false
	}
	r.updateHeartbeat()
	log.Printf("Entry appended. Heartbeat %v", r.state.LastHeartbeat)
	return true
}

func (r *Raft) resetElectionTimer() {
	log.Println("Reseting election timer")
	r.restartElectionTickerChannel <- 1
}

func (r *Raft) updateTermIfNeeded(term int) {
	if r.state.CurrentTerm < term {
		log.Printf("Stale term %v, updating to %v", r.state.CurrentTerm, term)
		r.state.CurrentTerm = term
		r.state.VotesGranted = 0
		r.state.VotedFor = ""
		r.convertToFollower()
	}
}

func (r *Raft) isLeader() bool {
	return r.state.Role == "leader"
}

func (r *Raft) isCandidate() bool {
	return r.state.Role == "candidate"
}

func (r *Raft) isFollower() bool {
	return r.state.Role == "follower"
}
