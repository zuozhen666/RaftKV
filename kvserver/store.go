package kvserver

import (
	"RaftKV/global"
	"errors"
	"log"
)

type KvStore struct {
	data     map[string]string
	proposeC chan<- global.Kv
	commitC  <-chan global.Kv
}

func NewKvStore(proposeC chan<- global.Kv, commitC <-chan global.Kv) *KvStore {
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
		switch commit.Op {
		case "put":
			kv.data[commit.Key] = commit.Val
		case "delete":
			delete(kv.data, commit.Key)
		}
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
