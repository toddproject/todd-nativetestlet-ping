package toddping

import (
	"errors"
	"fmt"
	"github.com/Mierdin/todd/agent/testing/testlets"
	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"net"
	"os"
	"runtime"
	"time"
)

type PingTestlet struct {
	testlets.BaseTestlet
}

func init() {

	var pt = PingTestlet{}

	// Ensure the RunFunction attribute is set correctly.
	// This allows the underlying testlet infrastructure
	// to know what function to call at runtime
	pt.RunFunction = pt.RunTestlet

	// This is important - register the name of this testlet
	// (the name the user will use in a testrun definition)
	testlets.Register("ping", &pt)
}

// RunTestlet implements the core logic of the testlet. Don't worry about running asynchronously,
// that's handled by the infrastructure.
func (p PingTestlet) RunTestlet(target string, args []string, kill chan (bool)) (map[string]string, error) {

	// Get number of pings
	count := 3 //TODO(mierdin): need to parse from 'args', or if omitted, use a default value

	log.Error(args)

	var latencies []float32
	var replies int

	// Execute ping once per count
	i := 0
	for i < count {
		select {
		case <-kill:
			// Terminating early; return empty metrics
			return map[string]string{}, nil
		default:

			//log.Debugf("Executing ping #%d", i)

			// Mocked ping logic
			latency, replyReceived := pingTemp(count)

			latencies = append(latencies, latency)

			if replyReceived {
				replies += 1
			}

			i += 1
			time.Sleep(1000 * time.Millisecond)

		}
	}

	// Calculate metrics
	var latencyTotal float32 = 0
	for _, value := range latencies {
		latencyTotal += value
	}
	avg_latency_ms := latencyTotal / float32(len(latencies))
	packet_loss := (float32(count) - float32(replies)) / float32(count)

	return map[string]string{
		"avg_latency_ms": fmt.Sprintf("%.2f", avg_latency_ms),
		"packet_loss":    fmt.Sprintf("%.2f", packet_loss),
	}, nil

}

func pingTemp(count int) (float32, bool) {
	return float32(count) * 4.234, true
}

// PingNative is a Go implementation of ping
// returns:
// float32 - response time in milliseconds
// bool - true if reply recieved before timeout
// error - nil if everything went well
func PingNative(ipv4Target string) (float32, bool, error) {

	// Detect v4/v6 here

	// Establish system compatbility
	switch runtime.GOOS {
	case "darwin":
	case "linux":
		log.Println("you may need to adjust the net.ipv4.ping_group_range kernel state")
	default:
		return 0, false, errors.New(fmt.Sprintf("ping testlet not supported on %s", runtime.GOOS))
	}

	// Start listening for response on all interfaces
	c, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	// Set the timeout to 3 seconds so the socket doesn't block forever,
	// but there's enough time for a reply
	// TODO(mierdin): Make this configurable via args
	c.SetReadDeadline(time.Now().Add(3 * time.Second))

	// Construct and send ICMP echo
	wm := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: 1,
			Data: []byte("hanshotfirst"),
		},
	}
	wb, err := wm.Marshal(nil)
	if err != nil {
		log.Error(err)
		return 0.0, false, nil
	}
	if _, err := c.WriteTo(wb, &net.UDPAddr{IP: net.ParseIP(ipv4Target)}); err != nil {
		log.Error(err)
		return 0.0, false, nil
	}

	// Is this the right place?
	start := time.Now()

	rb := make([]byte, 1500)
	n, peer, err := c.ReadFrom(rb)
	if err != nil {
		log.Debugf("Ping timeout on %v", peer)
		return 0.0, false, nil
	}

	// Is this the right place?
	elapsed := time.Since(start)

	// 1 is the protocol number for ICMP echo response. May need another block for 58, the IPv6 version
	rm, err := icmp.ParseMessage(1, rb[:n])
	if err != nil {
		log.Fatal(err)
	}

	switch rm.Type {
	case ipv6.ICMPTypeEchoReply:
	case ipv4.ICMPTypeEchoReply:
	default:
		log.Printf("ERROR. Got %+v; want echo reply", rm)
		return 0, false, errors.New("Received something other than an echo reply")
	}

	return float32(elapsed.Seconds() * 1e3), true, nil

}
