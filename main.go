// main.go

package main

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	logger "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func runSender(host string, ua string, dirPath string, worker int, deleteMode bool) {
	err := sender(host, ua, dirPath, worker, deleteMode)
	if err != nil {
		logger.Error("Client error, ", err.Error())
		return
	}
}

func runReciever(host string, ua string, dirpath string) {
	if dirpath == "" {
		saveFileDir, _ = filepath.Abs(filepath.Dir(os.Args[0]))
	} else {
		saveFileDir, _ = filepath.Abs(dirpath)
	}

	err := reciever(host, ua, dirpath)
	if err != nil {
		logger.Error("run server error")
		return
	}
}

func initLogger() {
	// logger.SetLevel(logger.DebugLevel)
	logger.SetFormatter(&logger.TextFormatter{
		FullTimestamp: true,
	})
}

func main() {
	initLogger()

	app := cli.NewApp()
	app.Name = "HTTPTransfer"
	app.Usage = "transfer files use HTTP"
	app.Version = "0.2-beta2"

	app.Commands = []cli.Command{
		{
			Name:    "recv",
			Aliases: []string{"r"},
			Usage:   "start transfer reciever",
			Action: func(c *cli.Context) error {
				if !c.IsSet("host") {
					logger.Error("must specify host to listen")
					return errors.New("must specify host to listen")
				}
				logger.Infof("reciever running on `%s`", c.String("host"))
				runReciever(c.String("host"), c.String("ua"), c.String("path"))
				return nil
			},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "host",
					Usage: "which host to listen",
				},
				cli.StringFlag{
					Name:  "path, p",
					Usage: "which path to save in",
				},
				cli.StringFlag{
					Name:  "ua, u",
					Value: "go-http_transfer",
					Usage: "User-Agent use in transfer",
				},
			},
		},
		{
			Name:    "send",
			Aliases: []string{"s"},
			Usage:   "start transfer sender",
			Action: func(c *cli.Context) error {
				deleteMode := false

				if !c.IsSet("host") {
					logger.Error("must specify host to send")
					return errors.New("must specify host to send")
				}
				if !c.IsSet("path") {
					logger.Error("must specify path to send")
					return errors.New("must specify path to send")
				}
				if c.IsSet("delete") {
					deleteMode = true
				}
				logger.Infof("sender will send `%s` to `%s`", c.String("path"), c.String("host"))
				runSender(c.String("host"), c.String("ua"), c.String("path"), c.Int("worker"), deleteMode)
				return nil
			},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "path, p",
					Usage: "which path to send",
				},
				cli.StringFlag{
					Name:  "host",
					Usage: "which host to send",
				},
				cli.StringFlag{
					Name:  "ua, u",
					Value: "go-http_transfer",
					Usage: "User-Agent use in transfer",
				},
				cli.IntFlag{
					Name:  "worker, w",
					Value: 2,
					Usage: "HTTP worker number",
				},
				cli.BoolFlag{
					Name:  "delete",
					Usage: "enable delete mode",
				},
			},
		},
	}

	app.Run(os.Args)
}
