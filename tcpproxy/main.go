package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"sync"
)

type confItem struct {
	From uint16
	To   uint16
}
type Config struct {
	Host     string
	BuffSize uint
	Items    []confItem
}

var BuffSize uint = 4 * 1024 * 1024 //4m
func main() {
	var conf Config
	confFile := filepath.Dir(os.Args[0]) + "/config.json"
	dat, err := ioutil.ReadFile(confFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			conf.Host = "192.168.12.36"
			conf.BuffSize = 4 * 1024
			conf.Items = append(conf.Items, confItem{
				From: 5700,
				To:   5700,
			})
			conf.Items = append(conf.Items, confItem{
				From: 5900,
				To:   5900,
			})
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
	if conf.BuffSize > 1024*1024*100 {
		fmt.Printf("buffer size too large: %v", conf.BuffSize)
		return
	}
	BuffSize = conf.BuffSize
	fmt.Printf("buffer size: %v\n", BuffSize)
	var wg sync.WaitGroup
	for _, item := range conf.Items {
		wg.Add(1)
		go func(fromPort, toPort uint16, toHost string) {
			defer wg.Done()
			proxyStart(fromPort, toPort, toHost)
		}(item.From, item.To, conf.Host)
	}
	wg.Wait()
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
	fmt.Printf("start proxy %v --> %v:%v\n", fromPort, toHost,toPort)
	for {
		proxyConn, err := proxyListener.Accept()
		if err != nil {
			fmt.Printf("Unable to accept a request, error: %s\n", err.Error())
			continue
		}

		// Read a header firstly in case you could have opportunity to check request
		// whether to decline or proceed the request
		buffer := make([]byte, 1024)
		n, err := proxyConn.Read(buffer)
		if err != nil {
			fmt.Printf("%v read error: %s\n",proxyConn.RemoteAddr(), err.Error())
			continue
		}

		// TODO
		// Your choice to make decision based on request header

		targetAddr := fmt.Sprintf("%s:%d", toHost, toPort)
		targetConn, err := net.Dial("tcp", targetAddr)
		if err != nil {
			fmt.Printf("Unable to connect to: %s, error: %s\n\n", targetAddr, err.Error())
			proxyConn.Close()
			continue
		}

		n, err = targetConn.Write(buffer[:n])
		if err != nil {
			fmt.Printf("Unable to write to output, error: %s\n", err.Error())
			proxyConn.Close()
			targetConn.Close()
			continue
		}
		fmt.Printf("new connection %v --> %v\n", proxyConn.RemoteAddr(),targetConn.RemoteAddr())
		go proxyRequest(proxyConn, targetConn)
		go proxyRequest(targetConn, proxyConn)
	}
}

// Forward all requests from r to w
func proxyRequest(r net.Conn, w net.Conn) {
	defer r.Close()
	defer w.Close()

	var buffer = make([]byte, BuffSize)
	for {
		n, err := r.Read(buffer)
		if err != nil {
			fmt.Printf("Unable to read from input, error: %s\n", err.Error())
			break
		}
		n, err = w.Write(buffer[:n])
		if err != nil {
			fmt.Printf("Unable to write to output, error: %s\n", err.Error())
			break
		}
	}
}
