package main

import "net/http"

func main() {
	httpServer := &http.Server{
		Addr: ":8080",
		Handler: &kvServer{
			store: NewKvStore(),
		},
	}
	httpServer.ListenAndServe()
}
