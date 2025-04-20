# Portal

## Experimental branch for HTTP termination


# Overview

The main goal of this project is to provide access from cloud to on-prem without opening ports on-prem. This library provides a mechanism to build a 2-node HTTP tunnel.


                   +---------+
                   | Cloud   |
                   | HTTPS   |
                   | Client  |
                   +----+----+
                        |
                        | HTTP/WS
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

Wrap the tunnel connection with Framer interface and use Serve:

    tn := portal.Tunnel{
        Client: // This http.Client for the tunnel client side only
    }
    tn.Serve(ctx, framer)

Framer interface is for reading and writing messages with boundaries (i.e. frame). The examples show a simple length/bytes and WebSocket framer.

For incoming HTTP/WS connections, pass the processing to the tunnel with:

    tn.ServeHTTP(w, r)

