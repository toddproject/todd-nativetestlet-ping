/*
   ToDD Client - Primary entrypoint

   Copyright 2016 Matt Oswalt. Use or modification of this
   source code is governed by the license provided here:
   https://github.com/Mierdin/todd/blob/master/LICENSE
*/

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"

	log "github.com/Sirupsen/logrus"
	cli "github.com/codegangsta/cli"

	"github.com/toddproject/todd-nativetestlet-ping/ping"
)

var (
	testletName = "ping"
)

func checkSystem() error {
	// Establish system compatbility
	switch runtime.GOOS {
	case "darwin":
	case "linux":
		log.Warn("Linux detected - please ensure that socket capabilities have been set")
	default:
		log.Error(fmt.Sprintf("'%s' testlet not supported on %s", testletName, runtime.GOOS))
		return errors.New("unsupported platform")
	}
	return nil
}

func check() error {
	err := checkSystem()
	if err != nil {
		os.Exit(1)
	}

	loopbacks := []string{
		"::1",
		"127.0.0.1",
	}

	successes := 0

	var pt = ping.PingTestlet{}
	for i := range loopbacks {
		metrics, err := pt.Run(loopbacks[i], map[string]interface{}{
			"count":       1,
			"icmpTimeout": 3,
		}, 1)
		if err != nil {
			log.Error("Problem sending test echo request: %v", err)
			continue
		}

		loss := float64(metrics["packet_loss"])
		if loss > 0.0 {
			log.Error("Unexpected packet loss on loopback")
			continue
		}
		successes++
	}

	if successes == 0 {
		return errors.New("Not enough successful pings. Check failed.")
	}

	return nil
}

func main() {

	app := cli.NewApp()
	app.Name = "toddping"
	app.Version = "v0.1.0"
	app.Usage = "A testlet for ICMP echos (ping)"

	var count, icmpTimeout int

	// global level flags
	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:        "c, count",
			Usage:       "number of pings to send",
			Value:       3,
			Destination: &count,
		},
		cli.IntFlag{
			Name:        "t, timeout",
			Usage:       "timeout for a single request",
			Value:       3,
			Destination: &icmpTimeout,
		},
	}

	// ToDD Commands
	app.Commands = []cli.Command{

		// "todd agents ..."
		{
			Name:  "check",
			Usage: "Show ToDD agent information",
			Action: func(c *cli.Context) {
				err := check()
				if err != nil {
					fmt.Println("Check mode FAILED")
					os.Exit(1)
				} else {
					fmt.Println("Check mode PASSED")
					os.Exit(0)
				}
			},
		},
	}

	app.Action = func(c *cli.Context) {

		var pt = ping.PingTestlet{}

		argMap := map[string]interface{}{
			"count":       count,
			"icmpTimeout": icmpTimeout,
		}

		metrics, err := pt.Run(os.Args[1], argMap, 30)
		if err != nil {
			errorMessage := fmt.Sprintf("Native testlet '%s' completed with error '%s'", testletName, err)
			log.Error(errorMessage)
			fmt.Println(errorMessage)
			//gatheredData[thisTarget] = "error"
			os.Exit(1)
		}

		// The metrics infrastructure requires that we collect metrics as a JSON string
		// (which is a result of building non-native testlets in early versions of ToDD)
		// So let's convert, and add to gatheredData
		metrics_json, err := json.Marshal(metrics)
		if err != nil {
			//TODO(mierdin) do something
		}
		//gatheredData[thisTarget] = string(metrics_json)
		fmt.Println(string(metrics_json))
	}

	app.Run(os.Args)
}
