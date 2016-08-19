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
	//"github.com/Mierdin/todd/agent/testing/testlets"
	"github.com/toddproject/todd-nativetestlet-ping/ping"
)

func main() {

	if os.Args[1] == "check" {
		fmt.Println("Check mode PASSED")
		os.Exit(0)
	}

	var pt = ping.PingTestlet{}

	// Ensure the RunFunction attribute is set correctly.
	// This allows the underlying testlet infrastructure
	// to know what function to call at runtime
	pt.RunFunction = pt.RunTestlet

	// This is important - register the name of this testlet
	// (the name the user will use in a testrun definition)
	//testlets.Register("ping", &pt)

	// nativeTestlet, err := testlets.NewTestlet(tr.Testlet)
	// if err != nil {
	// 	//TODO(mierdin) do something
	// }

	// metrics, err := nativeTestlet.Run("8.8.8.8", []string{"-c 10", "-s"}, ett.TimeLimit)
	// //log.Error(nativeTestlet.RunFunction)
	// if err != nil {
	// 	log.Errorf("Testlet <TESTLET> completed with error '%s'", err)
	// 	gatheredData[thisTarget] = "error"
	// }

	var testchan chan bool

	metrics, err := pt.RunTestlet(os.Args[1], os.Args[2:], testchan)
	if err != nil {
		//fmt.Errorf("Testlet <TESTLET> completed with error '%s'", err)
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
