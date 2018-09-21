package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

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
	http.Handle("/", logInfo(http.FileServer(http.Dir(directory)), directory))
	http.HandleFunc("/info/", func(w http.ResponseWriter, r *http.Request) {
		getImageInfo(w, r, directory)
	})
}

func logInfo(next http.Handler, directory string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timeStart := time.Now()
		clientID := "clientID: " + r.RemoteAddr
		log.Println(clientID + ", start: " + timeStart.Format(time.RFC3339))
		next.ServeHTTP(w, r)

		timeEnd := time.Now()
		log.Println(clientID + ", end: " + timeEnd.Format(time.RFC3339))

		imageName := strings.TrimPrefix(r.URL.Path, "/")
		if imageName == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("name can not be empty"))
			return
		}

		averageSpeed, err := countAverageSpeed(directory, imageName, timeEnd.Unix()-timeStart.Unix())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("error while calculating average speed, %v", err.Error())))
			return
		}
		log.Println(clientID+", average speed: ", averageSpeed/1000, "kB/s")
	})
}

//countAverageSpeed counts average speed of downloading image.
//it counts sizeOfImage(bytes)/durationOfDownload(seconds)
func countAverageSpeed(directory, imageName string, duration int64) (int64, error) {
	if _, ok := imagesInfo[imageName]; !ok {
		err := getInfo(directory, imageName)
		if err != nil {
			return 0, err
		}
	}

	data := make(map[string]interface{})
	err := json.Unmarshal(imagesInfo[imageName], &data)
	if err != nil {
		return 0, err
	}

	size := int64(data["virtual-size"].(float64))
	averageSpeed := size / duration
	return averageSpeed, nil
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

	err := getInfo(directory, imageName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(imagesInfo[imageName])
}

func getInfo(directory, imageName string) error {
	imagePath := directory + "/" + imageName
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return errors.New("image not found")
	}

	cmdName := "qemu-img"
	cmdArgs := []string{"info", imagePath, "--output=json"}
	cmd := exec.Command(cmdName, cmdArgs...)

	var (
		cmdOut []byte
		err    error
	)
	if cmdOut, err = cmd.Output(); err != nil {
		return fmt.Errorf("error while reading output of qemu-img command: %v", err.Error())
	}

	imagesInfo[imageName] = cmdOut
	return nil
}
