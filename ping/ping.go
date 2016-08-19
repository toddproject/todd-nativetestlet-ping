package ping

import (
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"
	"time"

	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"

	"github.com/Mierdin/todd/agent/testing/testlets"
)

type PingTestlet struct {
	testlets.BaseTestlet
}

// RunTestlet implements the core logic of the testlet. Don't worry about running asynchronously,
// that's handled by the infrastructure.
func (p PingTestlet) RunTestlet(target string, args []string, kill chan (bool)) (map[string]string, error) {

	// Get number of pings
	count := 3 //TODO(mierdin): need to parse from 'args', or if omitted, use a default value

	log.Error(target)
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

			// USE count

			// Mocked ping logic
			latency, replyReceived, _ := PingNative(target)
			//TODO handle err

			log.Errorf("Reply received after %f ms", latency)

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

// PingNative is a Go implementation of ping
// returns:
// float32 - response time in milliseconds
// bool - true if reply recieved before timeout
// error - nil if everything went well
func PingNative(target string) (float32, bool, error) {

	// Establish system compatbility
	switch runtime.GOOS {
	case "darwin":
	case "linux":
		log.Warn("Linux detected - you may need to adjust the net.ipv4.ping_group_range kernel state")
	default:
		return 0, false, errors.New(fmt.Sprintf("ping testlet not supported on %s", runtime.GOOS))
	}

	var proto, addy string
	var replyproto int

	// Detect v4/v6
	ip := net.ParseIP(target)
	if ip.To4() != nil {
		proto = "udp4"
		addy = "0.0.0.0"
		replyproto = 1
	} else {
		proto = "udp6"
		addy = "::"
		// replyproto = 129
		replyproto = 58
	}

	// Start listening for response on all interfaces
	c, err := icmp.ListenPacket(proto, addy)
	if err != nil {
		log.Fatal(err)
	}
	// time.Sleep(time.Second * 100000)
	defer c.Close()

	// Set the timeout to 3 seconds so the socket doesn't block forever,
	// but there's enough time for a reply
	// TODO(mierdin): Make this configurable via args
	c.SetReadDeadline(time.Now().Add(3 * time.Second))

	// Construct and send ICMP echo
	wm := icmp.Message{
		Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: 1,
			Data: []byte("hanshotfirst"),
		},
	}

	if ip.To4() != nil {
		wm.Type = ipv4.ICMPTypeEcho
	} else {
		wm.Type = ipv6.ICMPTypeEchoRequest
	}

	wb, err := wm.Marshal(nil)
	if err != nil {
		log.Error(err)
		return 0.0, false, nil
	}
	if _, err := c.WriteTo(wb, &net.UDPAddr{IP: net.ParseIP(target)}); err != nil {
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

	rm, err := icmp.ParseMessage(replyproto, rb[:n])
	if err != nil {
		log.Fatal(err)
	}

	switch rm.Type {
	case ipv4.ICMPTypeEchoReply:
		// This is an expected response type for ICMPv4 requests
	case ipv6.ICMPTypeEchoReply:
		// This is an expected response type for ICMPv6 requests, but....
	case ipv6.ICMPTypeEchoRequest:
		// ...this is what we end up seeing instead. TODO(mierdin): Dig into
		// this a bit more and see if this is a bug in the library
	default:
		log.Printf("ERROR. Got %+v; want echo reply", rm)
		return 0, false, errors.New("Received something other than an echo reply")
	}

	// Return the latency in milliseconds, and acknowledge that a reply was received
	return float32(elapsed.Seconds() * 1e3), true, nil

}
