package kvserver

import (
	"RaftKV/global"
	"errors"
	"log"
	"sync"
)

type KvStore struct {
	mu       sync.Mutex
	data     map[string]string
	proposeC chan<- global.Kv
	commitC  <-chan global.Commit
}

func NewKvStore(proposeC chan<- global.Kv, commitC <-chan global.Commit) *KvStore {
	kv := &KvStore{
		data:     make(map[string]string),
		proposeC: proposeC,
		commitC:  commitC,
	}
	go kv.readCommit()
	return kv
}

func (kv *KvStore) readCommit() {
	for commit := range kv.commitC {
		log.Printf("[kvserver]receive commit request %v from raft module", commit)
		kv.mu.Lock()
		switch commit.Kv.Op {
		case "put":
			kv.data[commit.Kv.Key] = commit.Kv.Val
		case "delete":
			delete(kv.data, commit.Kv.Key)
		}
		kv.mu.Unlock()
		commit.WG.Done()
	}
}

func (kv *KvStore) Put(key, value string) {
	propose := global.Kv{
		Key: key,
		Val: value,
		Op:  "put",
	}
	kv.proposeC <- propose
	log.Printf("[kvserver]propose %v to raft module", propose)
}

func (kv *KvStore) Get(key string) (string, error) {
	if v, ok := kv.data[key]; ok {
		return v, nil
	}
	return "", errors.New("Key Not Exist")
}

func (kv *KvStore) Delete(key string) {
	propose := global.Kv{
		Key: key,
		Op:  "delete",
	}
	kv.proposeC <- propose
	log.Printf("[kvserver]propose %v to raft module", propose)
}
