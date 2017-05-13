package ping

import (
	"errors"
	"net"
	"os"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"

	"github.com/Mierdin/todd/agent/testing"
)

type PingTestlet struct {
	testing.BaseTestlet
}

// RunTestlet implements the general workflow of the testlet. Lower-level functionality is implemented by the downstream function;
// this function focuses more on things like executing the right number of pings, and calculating metrics.
// timeout is a generic arg for all testlets (primarily for server-style testlets)
func (p PingTestlet) Run(target string, args map[string]interface{}, timeout int) (map[string]float32, error) {

	// Get args
	count := args["count"].(int)
	icmpTimeout := args["icmpTimeout"].(int)

	var latencies []float32
	var replies int

	// Execute ping once per count
	i := 0
	for i < count {

		latency, replyReceived, _ := PingNative(target, count, icmpTimeout)
		//TODO(mierdin): handle err

		if replyReceived {
			log.Infof("Reply received from %s after %f ms", target, latency)
		} else {
			log.Info("Request timed out.")
		}

		latencies = append(latencies, latency)

		if replyReceived {
			replies += 1
		}

		i += 1
		time.Sleep(1000 * time.Millisecond)
	}

	// Calculate metrics
	var latencyTotal float32 = 0
	for _, value := range latencies {
		latencyTotal += value
	}
	avg_latency_ms := latencyTotal / float32(len(latencies))
	packet_loss := (float32(count) - float32(replies)) / float32(count)

	// return map[string]string{
	// 	"avg_latency_ms": fmt.Sprintf("%.2f", avg_latency_ms),
	// 	"packet_loss":    fmt.Sprintf("%.2f", packet_loss),
	// }, nil

	return map[string]float32{
		"avg_latency_ms": avg_latency_ms,
		"packet_loss":    packet_loss,
	}, nil

}

// PingNative is a Go implementation of ping
// returns:
// float32 - response time in milliseconds
// bool - true if reply recieved before timeout
// error - nil if everything went well
func PingNative(target string, count, icmpTimeout int) (float32, bool, error) {

	var proto, addy string
	var requestproto, replyproto int

	// Detect v4/v6
	ip := net.ParseIP(target)

	// TODO(mierdin): Need to fall back on udp ping if the icmp approach doesn't work.
	// Was "udp4" and "udp6" before I decided to go the "capabilities" route

	if ip.To4() != nil {
		proto = "ip4:icmp"
		addy = "0.0.0.0"
		requestproto = 8
		replyproto = 1
	} else {
		// proto = "ip6:icmp"
		proto = "ip6:ipv6-icmp"
		addy = "::"
		// replyproto = 129
		requestproto = 128
		replyproto = 58
	}

	// Start listening for response on all interfaces
	// This will attempt a raw ICMP socket first, then fall back to UDP
	c, err := icmp.ListenPacket(proto, addy)
	if err != nil {
		if proto == "ip4:icmp" {
			proto = "udp4"
		} else if proto == "ip6:ipv6-icmp" {
			proto = "udp6"
		}
		c, err = icmp.ListenPacket(proto, addy)
		if err != nil {
			log.Error("Failed to open a socket. Please refer to the documentation for system compatibility")
			log.Fatal(err)
		}
	}

	log.Debugf("Opened %s socket", proto)

	// time.Sleep(time.Second * 100000)
	defer c.Close()

	// Set the timeout so the socket doesn't block forever
	// (default is 3 seconds)
	c.SetReadDeadline(time.Now().Add(time.Duration(icmpTimeout) * time.Second))

	log.Debug(requestproto)

	// Construct ICMP echo
	wm := icmp.Message{
		// Code: requestproto,
		Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: count,
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

	if strings.Contains(proto, "udp") {
		if _, err := c.WriteTo(wb, &net.UDPAddr{IP: net.ParseIP(target)}); err != nil {
			log.Error(err)
			return 0.0, false, nil
		}
	} else {
		if _, err := c.WriteTo(wb, &net.IPAddr{IP: net.ParseIP(target)}); err != nil {
			log.Error(err)
			return 0.0, false, nil
		}
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

	// This block performs triage on incoming requests
	//
	// TODO(mierdin): This has been a source of a lot of confusion. You will notice that
	// both requests and responses are in both the IPv4 and IPv6 blocks. This is because
	// rm.Type doesn't always seem to equal what's in the response packet (I checked with
	// (tcpdump)
	//
	// For instance:
	//
	// 	  case ipv6.ICMPTypeEchoReply:
	// 	  // This is an expected response type for ICMPv6 requests, and the packet contains
	//    // this code, but...
	//    case ipv6.ICMPTypeEchoRequest:
	// 	  // ...this is what we end up seeing instead, which is of course not the right code
	//    // for a response.
	//
	// Need to figure out what's causing this - it could be a bug in the library.
	if ip.To4() != nil {

		switch rm.Type {
		case ipv4.ICMPTypeEcho:
		case ipv4.ICMPTypeEchoReply:
			// This is an expected response type for ICMPv4 requests
		default:
			log.Printf("ERROR. Got %+v; want IPV4 echo reply", rm)
			return 0, false, errors.New("Received something other than an IPV4 echo reply")
		}
	} else {
		switch rm.Type {
		case ipv6.ICMPTypeEchoReply:
			// This is an expected response type for ICMPv6 requests, but....
		case ipv6.ICMPTypeEchoRequest:
			// ...this is what we end up seeing instead. TODO(mierdin): Dig into
			// this a bit more and see if this is a bug in the library
		default:
			log.Printf("ERROR. Got %+v; want IPV6 echo reply", rm)
			return 0, false, errors.New("Received something other than an IPV6 echo reply")
		}
	}

	// Return the latency in milliseconds, and acknowledge that a reply was received
	return float32(elapsed.Seconds() * 1e3), true, nil

}
