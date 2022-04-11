package raft

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type RaftServer struct {
	raft *Raft
}

func NewRaftServer(r *Raft) *RaftServer {
	return &RaftServer{
		raft: r,
	}
}

func (s *RaftServer) Start(listenAddr string) {
	http.HandleFunc("/raft/request-vote", s.HandleRequestVote)
	http.HandleFunc("/raft/append-entries", s.HandleAppendEntries)
	http.ListenAndServe(listenAddr, nil)
}

func (s *RaftServer) HandleRequestVote(writer http.ResponseWriter, request *http.Request) {
	requestVoteArgs := RequestVoteArgs{}
	body, _ := ioutil.ReadAll(request.Body)
	json.Unmarshal(body, &requestVoteArgs)
	requestVoteRes := s.raft.HandleRequestVote(requestVoteArgs)
	b, _ := json.Marshal(requestVoteRes)
	writer.Write(b)
	writer.Header().Add("content-type", "application/json")
}

func (s *RaftServer) HandleAppendEntries(writer http.ResponseWriter, request *http.Request) {
	appendEntriesArgs := AppendEntriesArgs{}
	body, _ := ioutil.ReadAll(request.Body)
	json.Unmarshal(body, &appendEntriesArgs)
	appendEntriesRes := s.raft.HandleAppendEntries(appendEntriesArgs)
	b, _ := json.Marshal(appendEntriesRes)
	writer.Write(b)
	writer.Header().Add("content-type", "application/json")
}
