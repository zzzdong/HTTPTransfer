// reciever.go

package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	logger "github.com/sirupsen/logrus"
)

var saveFileDir string
var gUserAgent string

func saveUploadFile(w http.ResponseWriter, r *http.Request) (err error) {
	ua := r.UserAgent()
	if ua != gUserAgent {
		err = errors.New("User-Agent not ok")
		return err
	}

	filename := r.Header.Get("File-Name")
	if filename == "" {
		err = errors.New("can not get File-Name")
		return err
	}
	filename = filepath.Join(saveFileDir, filename)

	// try to create file first
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		err = os.MkdirAll(filepath.Dir(filename), 0775)
		if err != nil {
			return err
		}
		file, err = os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			return err
		}
	}
	defer file.Close()

	_, err = io.Copy(file, r.Body)
	if err != nil {
		return err
	}

	logger.Info("saved file: ", filename)

	return nil
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	err := saveUploadFile(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func reciever(host string, ua string, dirpath string) (err error) {
	logger.Info("server will save file to ", saveFileDir)

	gUserAgent = ua

	err = os.MkdirAll(saveFileDir, 0775)
	if err != nil {
		return err
	}

	http.HandleFunc("/upload", uploadHandler)
	http.ListenAndServe(host, nil)

	return nil
}
