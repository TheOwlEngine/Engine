package main

import (
	"log"
	"net/http"
)

func main() {
	fs := http.FileServer(http.Dir("./pages"))
	http.Handle("/", fs)

	log.Print("Listening on http://localhost:8888")
	err := http.ListenAndServe(":8888", nil)

	if err != nil {
		log.Fatal(err)
	}
}
