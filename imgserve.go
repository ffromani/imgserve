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
	directory := flag.StringP("directory", "D", ".", "the directory of image files")
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
		log.Printf("%s start: %s", clientID, timeStart.Format(time.RFC3339))

		next.ServeHTTP(w, r)

		timeEnd := time.Now()
		log.Println("%s end: %s", clientID, timeEnd.Format(time.RFC3339))

		imageName := strings.TrimPrefix(r.URL.Path, "/")
		if imageName == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("name can not be empty"))
			return
		}

		duration := timeEnd.Unix() - timeStart.Unix()
		if duration <= 0 {
			// FIXME: explain how duration could possibly be < 0
			return
		}

		averageSpeed, err := countAverageSpeed(directory, imageName, duration)
		if err != nil {
			log.Printf("error while calculating average speed: %s", err.Error())
			return
		}
		log.Printf("%s average speed: %i kB/s", clientID, averageSpeed/1000)
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
	log.Printf("info about '%s'", imageName)

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
	log.Printf("getting QEMU info from '%s'", imagePath)

	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return errors.New("image not found")
	}

	cmdName := "qemu-img"
	cmdArgs := []string{"info", imagePath, "--output=json"}
	cmd := exec.Command(cmdName, cmdArgs...)

	if cmdOut, err := cmd.Output(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("error while reading output of qemu-img command: %v", err.Error())))
		return
	}

	imagesInfo[imageName] = cmdOut
	return nil
}
