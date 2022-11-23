# Simple tunnel example
The example runs tunnel on port 10001 and proxy on port 10002:

    # Run tunnel server
    simple-tunnel -server -tunnelAddress localhost:10001 -proxyAddress localhost:10002 

    # Run tunnel client
    simple-tunnel -client -tunnelAddress localhost:10001

Run HTTPS server on port 10003 and connect client via proxy port 10002:

    # Create https-server certificate for localhost
    openssl req -x509 -nodes -newkey rsa:2048 -sha256 -keyout https-server.key -out https-server.crt -subj "/C=US/CN=https-server" -extensions SAN -config <(cat /etc/ssl/openssl.cnf  <(printf "\n[SAN]\nsubjectAltName=DNS:localhost\n"))

    # Run HTTPS server with openssl
    openssl s_server -cert https-server.crt -key https-server.key -accept 10003 -www

    # Run HTTPS client with curl
    curl --proxy http://localhost:10002 --cacert https-server.crt https://localhost:10003
