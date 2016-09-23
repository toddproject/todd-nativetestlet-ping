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
	//cli "github.com/codegangsta/cli"

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

	var pt = ping.PingTestlet{}

	for i := range loopbacks {
		metrics, err := pt.Run(loopbacks[i], []string{""}, 1)

		loss := float64(metrics["packet_loss"])

		if err != nil {
			log.Error(err)
			return err
		}
		if loss > 0.0 {
			log.Error("packet loss on loopback")
			return errors.New("check failed")
		}
	}

	return nil
}

func main() {

	// Run this testlet's system check
	if os.Args[1] == "check" {
		err := check()
		if err != nil {
			fmt.Println("Check mode FAILED")
			os.Exit(1)
		} else {
			fmt.Println("Check mode PASSED")
			os.Exit(0)
		}
	}

	var pt = ping.PingTestlet{}

	// TODO accept timeout param
	metrics, err := pt.Run(os.Args[1], os.Args[2:], 30)
	if err != nil {
		errorMessage := fmt.Sprintf("Native testlet '%s' completed with error '%s'", testletName, err)
		log.Error(errorMessage)
		fmt.Println(errorMessage)
		//gatheredData[thisTarget] = "error"
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
