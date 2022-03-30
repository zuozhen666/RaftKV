package raft

import (
	"fmt"
	"math/rand"
	"time"
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

	election  chan bool
	heartbeat chan bool
}

func (r *Raft) broadcastRequestVote() {

}

func (r *Raft) broadcastHeartbeat() {

}

func (r *Raft) start() {
	r.state = Follower
	r.currentTerm = 0
	r.votedFor = -1
	r.election = make(chan bool)
	r.heartbeat = make(chan bool)

	go func() {

		for {
			switch r.state {
			case Follower:
				select {
				case <-r.heartbeat:
					fmt.Printf("follower-%d receive heartbeat\n", r.id)
				case <-time.After(time.Duration(rand.Intn(200)+300) * time.Millisecond):
					fmt.Printf("follower-%d election timeout\n", r.id)
					r.state = Candidate
				}
			case Candidate:
				r.currentTerm++
				r.votedFor = r.id
				go r.broadcastRequestVote()
				select {
				case <-r.election:
				case <-time.After(time.Duration(rand.Intn(200)+300) * time.Millisecond):
					fmt.Printf("candidate-%d election timeout\n", r.id)

				}
			case Leader:
				r.broadcastHeartbeat()
			}
		}
	}()
}
