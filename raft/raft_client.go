package raft

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

type RaftClient struct {
	httpClient http.Client
}

func NewRaftClient() *RaftClient {
	return &RaftClient{
		httpClient: http.Client{},
	}
}

func (c *RaftClient) AppendEntries(peer string, appendEntriesArgs AppendEntriesArgs) (AppendEntriesRes, error) {
	reqJson, _ := json.Marshal(appendEntriesArgs)
	res, err := c.httpClient.Post("http://"+peer+"/raft/append-entries", "application/json", bytes.NewBuffer(reqJson))
	if err != nil {
		log.Printf("[raft module]AppendEntries to Node %v http post error: %v", peer, err)
	}
	var appendEntriesRes = AppendEntriesRes{}
	if err != nil || res.StatusCode != 200 {
		return appendEntriesRes, err
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return appendEntriesRes, err
	}
	json.Unmarshal(body, &appendEntriesRes)
	return appendEntriesRes, nil
}

func (c *RaftClient) RequestVote(peer string, requestVoteArgs RequestVoteArgs) (RequestVoteRes, error) {
	reqJson, _ := json.Marshal(requestVoteArgs)
	res, err := c.httpClient.Post("http://"+peer+"/raft/request-vote", "application/json", bytes.NewBuffer(reqJson))
	if err != nil {
		log.Printf("[raft module]RequestVote to Node %v http post error: %v", peer, err)
	}
	var requestVoteRes = RequestVoteRes{}
	if err != nil || res.StatusCode != 200 {
		return requestVoteRes, err
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return requestVoteRes, err
	}
	json.Unmarshal(body, &requestVoteRes)
	return requestVoteRes, nil
}
