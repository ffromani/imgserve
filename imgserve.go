package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	flag "github.com/spf13/pflag"
)

var (
	imagesInfo = make(map[string][]byte)
)

func main() {
	port := flag.IntP("port", "p", 8001, "port to serve on")
	directory := flag.StringP("directory", "D", ".", "the directory of static file to host")
	flag.Parse()

	initRoutes(*directory)
	addr := fmt.Sprintf(":%d", *port)
	log.Fatal(http.ListenAndServe(addr, nil))
}

//initRoutes initializes all routes
func initRoutes(directory string) {
	http.Handle("/", http.FileServer(http.Dir(directory)))
	http.HandleFunc("/info/", func(w http.ResponseWriter, r *http.Request) {
		getImageInfo(w, r, directory)
	})
}

//getImageInfo returns info about image
func getImageInfo(w http.ResponseWriter, r *http.Request, directory string) {
	imageName := strings.TrimPrefix(r.URL.Path, "/info/")
	if imageName == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("name can not be empty"))
		return
	}

	if value, ok := imagesInfo[imageName]; ok {
		w.WriteHeader(http.StatusOK)
		w.Write(value)
		return
	}

	imagePath := directory + "/" + imageName
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("image does not exists"))
		return
	}

	cmdName := "qemu-img"
	cmdArgs := []string{"info", imagePath, "--output=json"}
	cmd := exec.Command(cmdName, cmdArgs...)

	var (
		cmdOut []byte
		err    error
	)
	if cmdOut, err = cmd.Output(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("error while reading output of qemu-img command: %v", err.Error())))
		return
	}

	imagesInfo[imageName] = cmdOut
	w.WriteHeader(http.StatusOK)
	w.Write(cmdOut)
}
