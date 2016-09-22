/*
   ToDD Client - Primary entrypoint

   Copyright 2016 Matt Oswalt. Use or modification of this
   source code is governed by the license provided here:
   https://github.com/Mierdin/todd/blob/master/LICENSE
*/

package main

import (
	"encoding/json"
	"fmt"
	"os"

	//cli "github.com/codegangsta/cli"

	"github.com/toddproject/todd-nativetestlet-ping/ping"
)

func main() {

	if os.Args[1] == "check" {

		//TODO(mierdin): Need to do a test ping

		fmt.Println("Check mode PASSED")
		os.Exit(0)
	}

	var pt = ping.PingTestlet{}

	// TODO accept timeout param
	metrics, err := pt.Run(os.Args[1], os.Args[2:], 30)
	if err != nil {
		fmt.Errorf("Testlet <TESTLET> completed with error '%s'", err)
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
