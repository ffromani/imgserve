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

// TODO: ensure thread safety
var (
	imagesInfo = make(map[string][]byte)
)

func main() {
	port := flag.IntP("port", "p", 8001, "port to serve on")
	directory := flag.StringP("directory", "D", ".", "the directory of image files")
	flag.Parse()

	initRoutes(*directory)

	// TODO: add option to pre-load infos about all the images found in the directory

	addr := fmt.Sprintf(":%d", *port)
	log.Fatal(http.ListenAndServe(addr, nil))
}

//initRoutes initializes all routes
func initRoutes(directory string) {
	http.Handle("/", downloadSpeedHandler(http.FileServer(http.Dir(directory)), directory))
	http.HandleFunc("/info/", func(w http.ResponseWriter, r *http.Request) {
		infoHandler(w, r, directory)
	})
}

func downloadSpeedHandler(next http.Handler, directory string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timeStart := time.Now()
		clientID := "clientID: " + r.RemoteAddr
		log.Printf("%s download BEGIN", clientID)

		next.ServeHTTP(w, r)

		duration := time.Now().Sub(timeStart)
		log.Printf("%s download FINISH in %v", clientID, time.Duration(duration))

		logDownloadSpeed(clientID, r.URL.Path, directory, int64(duration.Seconds()))
	})
}

func logDownloadSpeed(clientID string, urlPath string, directory string, duration int64) {
	if duration <= 0 {
		log.Printf("%s: unexpected duration %d (no average speed reported)", clientID, duration)
		return
	}

	imageName := strings.TrimPrefix(urlPath, "/")
	if imageName == "" {
		log.Printf("%s: unrecognized image name from '%s' (no average speed reported)", clientID, urlPath)
		return
	}

	averageSpeed, err := countAverageSpeed(directory, imageName, duration)
	if err != nil {
		log.Printf("%s: %s (no average speed reported)", clientID, err.Error())
		return
	}

	log.Printf("%s average speed: %d kB/s", clientID, averageSpeed/1000)

}

//countAverageSpeed counts average speed of downloading image.
//it counts sizeOfImage(bytes)/durationOfDownload(seconds)
func countAverageSpeed(directory, imageName string, duration int64) (int64, error) {
	imageSpec, ok := imagesInfo[imageName]
	if !ok {
		return 0, errors.New(fmt.Sprintf("No info about image '%s'", imageName))
	}

	data := make(map[string]interface{})
	err := json.Unmarshal(imageSpec, &data)
	if err != nil {
		return 0, err
	}

	size := int64(data["virtual-size"].(float64))
	averageSpeed := size / duration
	return averageSpeed, nil
}

//infoHandler returns info about image
func infoHandler(w http.ResponseWriter, r *http.Request, directory string) {
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

	err := getQEMUImageInfo(directory, imageName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(imagesInfo[imageName])
}

func getQEMUImageInfo(directory, imageName string) error {
	imagePath := directory + "/" + imageName
	log.Printf("getting QEMU info from '%s'", imagePath)

	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return errors.New("image not found")
	}

	cmdName := "qemu-img"
	cmdArgs := []string{"info", imagePath, "--output=json"}
	cmd := exec.Command(cmdName, cmdArgs...)

	cmdOut, err := cmd.Output()
	if err != nil {
		return errors.New(fmt.Sprintf("error while reading output of qemu-img command: %v", err.Error()))
	}

	imagesInfo[imageName] = cmdOut
	return nil
}
