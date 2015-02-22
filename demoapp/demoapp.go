package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

var port = flag.Int("port", 8080, "port")

func main() {
	flag.Parse()
	fmt.Println("Listening on port", *port)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		//w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Hello from %d", *port)
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
