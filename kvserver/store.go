package kvserver

import (
	"RaftKV/config"
	"errors"
)

type KvStore struct {
	data     map[string]string
	proposeC chan<- config.Kv
	commitC  <-chan config.Kv
}

func NewKvStore(proposeC chan<- config.Kv, commitC <-chan config.Kv) *KvStore {
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
		switch commit.Op {
		case "put":
			kv.data[commit.Key] = commit.Val
		case "delete":
			delete(kv.data, commit.Key)
		}
	}
}

func (kv *KvStore) Put(key, value string) {
	kv.proposeC <- config.Kv{
		Key: key,
		Val: value,
		Op:  "put",
	}
}

func (kv *KvStore) Get(key string) (string, error) {
	if v, ok := kv.data[key]; ok {
		return v, nil
	}
	return "", errors.New("Key Not Exist")
}

func (kv *KvStore) Delete(key string) {
	kv.proposeC <- config.Kv{
		Key: key,
		Op:  "delete",
	}
}
