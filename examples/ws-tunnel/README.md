# WebSocket tunnel example
The example runs both websocket tunnel and proxy on port 10001. Both tunnel and proxy connections are protected by TLS and bearer tokens:

    # Create tunnel-server certificate for localhost
    openssl req -x509 -nodes -newkey rsa:2048 -sha256 -keyout tunnel-server.key -out tunnel-server.crt -subj "/C=US/CN=tunnel-server" -extensions SAN -config <(cat /etc/ssl/openssl.cnf  <(printf "\n[SAN]\nsubjectAltName=DNS:localhost\n"))

    # Run tunnel server
    ws-tunnel -server -address :10001 -tunnelBearerAuth token1 -proxyBearerAuth token2 -cert tunnel-server.crt -key tunnel-server.key 

    # Run tunnel client
    ws-tunnel -client -address localhost:10001 -tunnelBearerAuth token1 -trust tunnel-server.crt


Run HTTPS server on port 10003 and connect client via proxy port 10001:

    # Create https-server certificate for localhost
    openssl req -x509 -nodes -newkey rsa:2048 -sha256 -keyout https-server.key -out https-server.crt -subj "/C=US/CN=https-server" -extensions SAN -config <(cat /etc/ssl/openssl.cnf  <(printf "\n[SAN]\nsubjectAltName=DNS:localhost\n"))

    # Run HTTPS server with openssl
    openssl s_server -cert https-server.crt -key https-server.key -accept 10003 -www

    # Run HTTPS client with curl
    curl --proxy https://localhost:10001 --proxy-cacert tunnel-server.crt --proxy-header "Proxy-Authorization: Bearer token2" --cacert https-server.crt https://localhost:10003
