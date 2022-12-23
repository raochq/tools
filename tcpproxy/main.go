package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"sync"
)

type Port struct {
	From uint16
	To   uint16
}
type Proxy struct {
	Host  string
	Ports []Port
}

type Config []Proxy

func main() {
	var conf Config
	confFile := filepath.Dir(os.Args[0]) + "/config.json"
	dat, err := ioutil.ReadFile(confFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			var proxy Proxy
			proxy.Host = "192.168.100.132"
			proxy.Ports = append(proxy.Ports,
				Port{
					From: 5500,
					To:   5500,
				},
				Port{
					From: 5700,
					To:   5700,
				},
				Port{
					From: 5900,
					To:   5900,
				})
			conf = []Proxy{proxy}
			dat, err = json.MarshalIndent(conf, "", "  ")
			if err == nil {
				ioutil.WriteFile(confFile, dat, os.ModePerm)
			}
		}
		fmt.Println(err)
		os.Exit(1)
	}
	err = json.Unmarshal(dat, &conf)
	if err != nil {
		fmt.Printf("json Unmarshal error: %v", err)
		return
	}
	var wg sync.WaitGroup
	for _, item := range conf {
		ProxyItem(&wg, item)
	}
	wg.Wait()
}

func ProxyItem(wg *sync.WaitGroup, conf Proxy) {
	for i := range conf.Ports {
		wg.Add(1)
		item := conf.Ports[i]
		go func() {
			defer wg.Done()
			proxyStart(item.From, item.To, conf.Host)
		}()
	}
}

// Start a proxy server listen on fromPort
// this proxy will then forward all request from fromPort to toPort
//
// Notice: a service must has been started on toPort
func proxyStart(fromPort, toPort uint16, toHost string) {
	proxyaddr := fmt.Sprintf(":%d", fromPort)
	proxyListener, err := net.Listen("tcp", proxyaddr)
	if err != nil {
		fmt.Printf("Unable to listen on: %s, error: %s\n", proxyaddr, err.Error())
		os.Exit(1)
	}
	defer proxyListener.Close()
	fmt.Printf("start proxy %v --> %v:%v\n", fromPort, toHost, toPort)
	for {
		proxyConn, err := proxyListener.Accept()
		if err != nil {
			fmt.Printf("Unable to accept a request, error: %s\n", err.Error())
			continue
		}
		go handle(proxyConn, fmt.Sprintf("%s:%d", toHost, toPort))
	}
}
func handle(src net.Conn, targetAddr string) {
	defer src.Close()

	dest, err := net.Dial("tcp", targetAddr)
	if err != nil {
		fmt.Printf("Unable to connect to: %s, error: %s\n\n", targetAddr, err.Error())
		return
	}
	defer dest.Close()

	fmt.Printf("new connection %v --> %v\n", src.RemoteAddr(), dest.RemoteAddr())
	defer fmt.Printf("close %v --> %v\n", src.RemoteAddr(), dest.RemoteAddr())

	exitchan := make(chan bool, 1)

	go func() {
		_, err := io.Copy(dest, src)
		fmt.Println(err)
		exitchan <- true
	}()

	go func() {
		_, err := io.Copy(src, dest)
		fmt.Println(err)
		exitchan <- true
	}()

	<-exitchan
}
