// sender.go

package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/karrick/godirwalk"
	"github.com/pkg/errors"
	logger "github.com/sirupsen/logrus"
	"golang.org/x/sync/syncmap"
)

var basePath string
var fileQueue = make(chan string, 128)
var fileList syncmap.Map
var oneShot = true
var sendIdle = make(chan bool, 1)
var gDeleteMode = false
var gSendUA string
var scanInterval = 5 * time.Second

func postFile(client *http.Client, url string, filePath string) (err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	filename, err := filepath.Rel(basePath, filePath)
	if err != nil {
		return errors.Errorf("got relative path failed, %s", err)
	}

	req, err := http.NewRequest("POST", url, file)
	req.Header.Set("User-Agent", gSendUA)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Add("File-Name", filename)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.Errorf("HTTP return %d", resp.StatusCode)
	}

	logger.Infof("sent: %s", filePath)

	return nil
}

func loopSendfiles(url string) {

	client := &http.Client{}

	for {
		select {
		case filename := <-fileQueue:
			for {
				err := postFile(client, url, filename)
				if err != nil {
					logger.Errorf("send %s failed, error: %s", filename, err)
					// if file is not exist, drop it
					if os.IsNotExist(err) {
						fileList.Delete(filename)
						break
					} else {
						logger.Debug("send failed, sleep then retry")
						time.Sleep(1 * time.Second)
					}
				} else {
					fileList.Delete(filename)
					if gDeleteMode {
						os.Remove(filename)
					}
					break
				}
			}
		}
	}
}

func getDirFiles(dirPath string) (err error) {

	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		logger.Errorf("read directory %s error, %s", dirPath, err)
		return errors.Wrap(err, "read directory error")
	}
	for _, info := range files {
		filename, _ := filepath.Abs(filepath.Join(dirPath, info.Name()))
		if time.Since(info.ModTime()) < 10*time.Second {
			continue
		}
		if _, ok := fileList.Load(filename); ok == false {
			logger.Debugf(" %s not in map", filename)
			fileList.Store(filename, true)
			fileQueue <- filename
		}
	}

	return nil
}

func travelDir(path string, de *godirwalk.Dirent) (err error) {
	if de.IsRegular() {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}

		if time.Since(info.ModTime()) < 10*time.Second {
			return nil
		}
		// var fileName = filepath.Join(path, info.Name())
		var fileName = path

		logger.Debugf("found: %s", fileName)
		if _, ok := fileList.Load(fileName); ok == false {
			fileList.Store(fileName, true)
			fileQueue <- fileName
		}
	}

	return nil
}

func walkDirFiles(dir string) (err error) {
	err = godirwalk.Walk(dir, &godirwalk.Options{
		Callback: travelDir,
		ErrorCallback: func(osPathname string, err error) godirwalk.ErrorAction {
			// Your program may want to log the error somehow.
			logger.Errorf("godirwalk.Walk error: %s", err)

			// For the purposes of this example, a simple SkipNode will suffice,
			// although in reality perhaps additional logic might be called for.
			return godirwalk.SkipNode
		},
	})
	if err != nil {
		logger.Errorf("walk directory %s failed, error: %s", dir, err)
	}

	return err
}

func sender(host string, ua string, dirPath string, workerNum int, deleteMode bool) (err error) {
	var url = "http://" + host + "/upload"

	gDeleteMode = deleteMode
	gSendUA = ua

	basePath, err = filepath.Abs(dirPath)
	if err != nil {
		return errors.Errorf("dirPath %s not ok, error: %s", dirPath, err)
	}

	info, err := os.Stat(basePath)
	if err != nil {
		return errors.Errorf("dirPath %s not ok, error: %s", dirPath, err)
	}
	if !info.IsDir() {
		return errors.Errorf("dirPath %s not a directory", basePath)
	}

	for i := 0; i < workerNum; i++ {
		go loopSendfiles(url)
	}

	lastScan := time.Now()
	// do scan at first time
	walkDirFiles(basePath)

	for {
		// let get file tree
		now := time.Now()
		if now.Sub(lastScan) > scanInterval {
			walkDirFiles(basePath)
			lastScan = now
		} else {
			time.Sleep(1 * time.Second)
		}
	}
}
