package main

import (
	"io"
	"log"
	"net"
	"net/http"

	"github.com/armon/go-socks5"
)

func handleConnect(w http.ResponseWriter, req *http.Request) {
	destConn, err := net.Dial("tcp", req.Host)
	if err != nil {
		http.Error(w, "Unable to connect to destination", http.StatusServiceUnavailable)
		return
	}
	defer destConn.Close()

	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, "Hijacking failed", http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	go io.Copy(destConn, clientConn)
	io.Copy(clientConn, destConn)
}

func requestHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodConnect {
		handleConnect(w, req)
		return
	}

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		http.Error(w, "Error when processing request", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func startHTTPProxy() {
	http.HandleFunc("/", requestHandler)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Error in HTTP ListenAndServe: %v", err)
	}
}

func startSOCKSProxy() {
	conf := &socks5.Config{}
	server, err := socks5.New(conf)
	if err != nil {
		log.Fatalf("Error creating SOCKS5 server: %v", err)
	}

	if err := server.ListenAndServe("tcp", ":8085"); err != nil {
		log.Fatalf("Error in SOCKS5 ListenAndServe: %v", err)
	}
}

func main() {
	go startHTTPProxy()
	startSOCKSProxy()
}
