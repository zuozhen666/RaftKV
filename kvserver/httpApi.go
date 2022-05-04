package kvserver

import (
	"io"
	"net/http"
)

type KvServer struct {
	Store *KvStore
}

func (kv *KvServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.RequestURI
	defer r.Body.Close()
	switch r.Method {
	case http.MethodPut:
		v, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed on PUT", http.StatusBadRequest)
			return
		}
		kv.Store.Put(key, string(v))
	case http.MethodGet:
		if v, err := kv.Store.Get(key); err == nil {
			w.Write([]byte(v + "\n"))
		} else {
			http.Error(w, "Failed to GET", http.StatusNotFound)
		}
	case http.MethodDelete:
		kv.Store.Delete(key)
	default:
		w.Header().Set("Allow", http.MethodPut)
		w.Header().Set("Allow", http.MethodGet)
		w.Header().Set("Allow", http.MethodDelete)
		http.Error(w, "Method nod Allowed", http.StatusMethodNotAllowed)
	}
}
