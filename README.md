# Portal

[![Go Reference](https://pkg.go.dev/badge/github.com/oatcode/portal.svg)](https://pkg.go.dev/github.com/oatcode/portal)
[![Release](https://img.shields.io/github/v/release/oatcode/portal)](https://github.com/oatcode/portal/releases)


A Go implementation of HTTP tunneling through a tunnel

# Overview

The main goal of this project is to provide access from cloud to on-prem without opening ports on-prem. This library provides a mechanism to build a 2-node HTTP tunnel.

The tunnel has two sides: client and server.
An on-prem application running tunnel client will connect to tunnel server running in cloud. Proxy port can be opened on cloud side to allow access to on-prem via HTTP tunnelling: <https://en.wikipedia.org/wiki/HTTP_tunnel>

This library only supports HTTPS tunneling that uses HTTP CONNECT to initiate connection.

                   +---------+
                   | Cloud   |
                   | HTTPS   |
                   | Client  |
                   +----+----+
                        |
                        | proxy
                +-------v-------+
                |               |
                | Tunnel Server |
                |     using     |
                |Portal library |
                |               |
                +-----+---^-----+
     Internet         |   |
    ------------------+---+--------------------
     On-prem          |   |
                +-----v---+-----+
                |               |
                | Tunnel Client |
                |     using     |
                |Portal library |
                |               |
                +-------+-------+
                        |
                        |
                   +----v----+
                   | On-prem |
                   | HTTPS   |
                   | Server  |
                   +---------+


# Install

    go get github.com/oatcode/portal

# Usage

Wrap the tunnel connection with Framer interface and use TunnelServe:

    coch := make(chan portal.ConnectOperation)
    portal.TunnelServe(ctx, framer, coch)

Framer interface is for reading and writing messages with boundaries (i.e. frame). The examples show a simple length/bytes and WebSocket framer.

coch is the channel to handle incoming proxy connection. Fill the ConnectOperation struct with net.Conn and proxy connect address. The examples illustrate how this is done with Go's http Hijack function.


