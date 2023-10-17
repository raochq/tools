package main

import (
	"fmt"
	"io"
	"net"
	"sync"
)

type Port struct {
	From uint16 `json:"from"`
	To   uint16 `json:"to"`
}
type TaskConf struct {
	Name  string `json:"title"`
	Host  string `json:"host"`
	Ports []Port `json:"ports"`
}
type ProxyTask struct {
	TaskConf
	stop    chan struct{}
	onClose func()
}

func NewProxyTask(name string, host string, ports []Port) *ProxyTask {
	return &ProxyTask{
		TaskConf: TaskConf{
			Name:  name,
			Host:  host,
			Ports: ports,
		},
		stop: make(chan struct{}),
	}
}

func (conf *ProxyTask) Start(wg *sync.WaitGroup, onClose func()) error {
	conf.onClose = onClose
	for i := range conf.Ports {
		item := conf.Ports[i]
		l, err := conf.listen(item.From)
		if err != nil {
			conf.Stop()
			return err
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			conf.proxyStart(l, item.From, item.To, conf.Host)
		}()
	}
	return nil
}
func (conf *ProxyTask) Stop() {
	close(conf.stop)
	if conf.onClose != nil {
		conf.onClose()
	}
}

func (conf *ProxyTask) listen(fromPort uint16) (net.Listener, error) {
	proxyaddr := fmt.Sprintf(":%d", fromPort)
	proxyListener, err := net.Listen("tcp", proxyaddr)
	if err != nil {
		fmt.Printf("Unable to listen on: %s, error: %s\n", proxyaddr, err.Error())
		return nil, err
	}
	return proxyListener, nil
}
func (conf *ProxyTask) proxyStart(listener net.Listener, fromPort, toPort uint16, toHost string) {
	fmt.Printf("start proxy %v --> %v:%v\n", fromPort, toHost, toPort)
	go func() {
		<-conf.stop
		listener.Close()
	}()
	for {
		proxyConn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Unable to accept a request, error: %s\n", err.Error())
			break
		}
		go conf.handle(proxyConn, fmt.Sprintf("%s:%d", toHost, toPort))
	}
}
func (conf *ProxyTask) handle(src net.Conn, targetAddr string) {
	defer src.Close()

	dest, err := net.Dial("tcp", targetAddr)
	if err != nil {
		fmt.Printf("Unable to connect to: %s, error: %s\n\n", targetAddr, err.Error())
		return
	}
	defer dest.Close()

	fmt.Printf("new connection %v --> %v\n", src.RemoteAddr(), dest.RemoteAddr())
	defer fmt.Printf("close %v --> %v\n", src.RemoteAddr(), dest.RemoteAddr())

	exitchan := make(chan bool, 2)

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
	for {
		select {
		case <-exitchan:
		case <-conf.stop:
			src.Close()
			dest.Close()
		}
	}
}
