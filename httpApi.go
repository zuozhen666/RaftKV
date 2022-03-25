package main

import (
	"io"
	"net/http"
)

type kvServer struct {
	store *kvstore
}

func (kv *kvServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.RequestURI
	defer r.Body.Close()
	switch r.Method {
	case http.MethodPut:
		v, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed on PUT", http.StatusBadRequest)
			return
		}
		kv.store.Put(key, string(v))
	case http.MethodGet:
		if v, err := kv.store.Get(key); err == nil {
			w.Write([]byte(v + "\n"))
		} else {
			http.Error(w, "Failed to GET", http.StatusNotFound)
		}
	case http.MethodDelete:
		if v, err := kv.store.Delete(key); err == nil {
			w.Write([]byte(v + "\n"))
		} else {
			http.Error(w, "Key not Exist", http.StatusBadRequest)
		}
	default:
		w.Header().Set("Allow", http.MethodPut)
		w.Header().Set("Allow", http.MethodGet)
		w.Header().Set("Allow", http.MethodDelete)
		http.Error(w, "Method nod Allowed", http.StatusMethodNotAllowed)
	}
}
