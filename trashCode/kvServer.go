package main

import (
	"io"
	"net/http"
)

type kvServer struct {
	store map[string]string
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
		kv.store[key] = string(v)
		w.WriteHeader(http.StatusNoContent)
	case http.MethodGet:
		if v, ok := kv.store[key]; ok {
			w.Write([]byte(v))
		} else {
			http.Error(w, "Failed to GET", http.StatusNotFound)
		}
	case http.MethodDelete:
		if v, ok := kv.store[key]; ok {
			w.Write([]byte(v))
			delete(kv.store, key)
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

func main() {
	httpServer := http.Server{
		Addr: ":8080",
		Handler: &kvServer{
			store: make(map[string]string),
		},
	}
	httpServer.ListenAndServe()
}
