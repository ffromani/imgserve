package main

import (
	"fmt"
	"log"
	"net/http"

	flag "github.com/spf13/pflag"
)

func main() {
	port := flag.IntP("port", "p", 8001, "port to serve on")
	directory := flag.StringP("directory", "D", ".", "the directory of static file to host")
	flag.Parse()

	addr := fmt.Sprintf(":%d", *port)

	log.Fatal(http.ListenAndServe(addr, http.FileServer(http.Dir(*directory))))
}
