package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/user"
	"strconv"
)

var portFlag = flag.Int("port", 8080, "port")

func main() {
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	if port == 0 {
		flag.Parse()
		port = *portFlag
	}
	fmt.Println("Listening on port", port)

	fmt.Println("Environment:")
	for _, e := range os.Environ() {
		fmt.Println(e)
	}

	u, _ := user.Current()
	fmt.Println("User:", u.Name)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		//w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Hello from %d", port)
		fmt.Printf("New request to %d\n", port)
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
