package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

func main() {
	var address string
	var certFile string
	var keyFile string
	flag.StringVar(&address, "address", "", "Address [<server-ip>]:<server-port>")
	flag.StringVar(&certFile, "cert", "", "TLS certificate filename")
	flag.StringVar(&keyFile, "key", "", "TLS certificate key filename")
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello url=%s", r.URL)
	})
	log.Fatal(http.ListenAndServeTLS(address, certFile, keyFile, nil))
}
