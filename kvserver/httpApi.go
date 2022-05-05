package kvserver

import (
	"RaftKV/global"
	"io"
	"net/http"
)

type KvServer struct {
	Store *KvStore
}

func (kv *KvServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if global.Node.KvPort != global.ClusterMeta.LeaderKvPort {
		http.Error(w, "please request leader node: localhost"+global.ClusterMeta.LeaderKvPort, http.StatusBadRequest)
		return
	}
	key := r.RequestURI
	defer r.Body.Close()
	switch r.Method {
	case http.MethodPut:
		v, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "[kvserver]Failed on PUT", http.StatusBadRequest)
			return
		}
		kv.Store.Put(key, string(v))
	case http.MethodGet:
		if v, err := kv.Store.Get(key); err == nil {
			w.Write([]byte(v + "\n"))
		} else {
			http.Error(w, "[kvserver]Failed to GET", http.StatusNotFound)
		}
	case http.MethodDelete:
		kv.Store.Delete(key)
	default:
		w.Header().Set("Allow", http.MethodPut)
		w.Header().Set("Allow", http.MethodGet)
		w.Header().Set("Allow", http.MethodDelete)
		http.Error(w, "[kvserver]Method nod Allowed", http.StatusMethodNotAllowed)
	}
}
