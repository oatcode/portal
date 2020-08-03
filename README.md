# Portal

A Go implementation of HTTP tunneling through a tunnel

## Overview

The main goal of this project is to provide access from cloud to on-prem without opening ports on-prem. This library provides a mechanism to build a 2-node HTTP tunnel.

The tunnel has two sides: client and server.
An on-prem application running tunnel client will connect to tunnel server running in cloud. Proxy port can be opened on cloud side to allow access to on-prem via HTTP tunnelling: <https://en.wikipedia.org/wiki/HTTP_tunnel>

This library only supports HTTPS tunneling that uses HTTP CONNECT to initiate connection.

## Usage

Establish the connection c and use:

    cch := make(chan net.Conn)
    portal.TunnelServe(c, cch)

where cch is the channel to handle incoming proxy connection

## Examples

Included in the projects are example code to establish a TLS tunnel and make HTTPS connection through it.

Create certificates for tunnel and https server:

    openssl req -x509 -nodes -newkey rsa:2048 -sha256 -keyout tunnel-server.key -out tunnel-server.crt -subj "/C=US/ST=CA/L=SJC/O=Example/OU=Dev/CN=localhost"
    openssl req -x509 -nodes -newkey rsa:2048 -sha256 -keyout tunnel-client.key -out tunnel-client.crt -subj "/C=US/ST=CA/L=SJC/O=Example/OU=Dev/CN=client"
    openssl req -x509 -nodes -newkey rsa:2048 -sha256 -keyout https-server.key -out https-server.crt -subj "/C=US/ST=CA/L=SJC/O=Example/OU=Dev/CN=localhost"

Running TLS tunnel client and server on port 10001, where proxy is on TLS tunnel server side on port 10002:

    tunnel-server -address :10001 -proxy :10002 -cert tunnel-server.crt -key tunnel-server.key -trust tunnel-client.crt
    tunnel-client -address localhost:10001 -cert tunnel-client.crt -key tunnel-client.key -trust tunnel-server.crt

Run HTTPS client and server on port 10003

    sample-https-server -address :10003 -cert https-server.crt -key https-server.key
    sample-https-client -proxy http://localhost:10002 -url https://localhost:10003/test -trust https-server.crt 

## Other ways to set proxy

The sample-https-client sets proxy programmatically. But it can be set in other ways. For example:

- export https_proxy=[proxy-host]:[proxy-port]
- java -Dhttps.proxyHost=[proxy-host] -Dhttps.proxyPort=[proxy-port]
