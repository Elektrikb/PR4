package main

import (
	"fmt"
	"game/handlers"
	"log"
	"net/http"
)

func main() {
	handler := handlers.NewHandler()

	http.HandleFunc("/join", handler.Join)
	http.HandleFunc("/guess", handler.Guess)
	http.HandleFunc("/status", handler.Status)

	fmt.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
