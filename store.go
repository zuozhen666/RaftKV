package main

import "errors"

type kvstore struct {
	m map[string]string
}

func NewKvStore() *kvstore {
	return &kvstore{
		m: make(map[string]string),
	}
}

func (kv *kvstore) Put(key, value string) {
	kv.m[key] = value
}

func (kv *kvstore) Get(key string) (string, error) {
	if v, ok := kv.m[key]; ok {
		return v, nil
	}
	return "", errors.New("Key Not Exist")
}

func (kv *kvstore) Delete(key string) (string, error) {
	if v, ok := kv.m[key]; ok {
		delete(kv.m, key)
		return v, nil
	}
	return "", errors.New("Key Not Exist")
}
