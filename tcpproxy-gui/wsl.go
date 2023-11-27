package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
	"unicode/utf16"
	"unsafe"
)

type routeInfo struct {
	net  uint32
	mask uint32
}

func hexToUint32LE(hex string) (uint32, error) {
	i, err := strconv.ParseInt(hex[6:8]+hex[4:6]+hex[2:4]+hex[0:2], 16, 64)
	if err != nil {
		return 0, err
	}
	return uint32(i), nil
}

func getRouteInfo(name string) (*routeInfo, error) {
	cmd := exec.Command("wsl.exe", "-d", name, "--", "cat", "/proc/net/route")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	ri := &routeInfo{}
	sout := string(out)
	sout = strings.TrimSpace(sout)
	lines := strings.Split(sout, "\n")
	lines = lines[1:]
	for _, line := range lines {
		fs := strings.Fields(line)
		if ri.mask > 0 && ri.net > 0 {
			break
		}
		if fs[0] != "eth0" {
			continue
		}
		if fs[1] != "00000000" {
			net, err := hexToUint32LE(fs[1])
			if err != nil {
				return nil, fmt.Errorf("failed to convert network to Uint32: %w", err)
			}
			ri.net = net
		}
		if fs[7] != "00000000" {
			mask, err := hexToUint32LE(fs[7])
			if err != nil {
				return nil, fmt.Errorf("failed to convert netmask to Uint32: %w", err)
			}
			ri.mask = mask
		}
	}

	return ri, nil
}

func isIPInRange(ri *routeInfo, ip uint32) bool {
	return (ri.net & ri.mask) == (ip & ri.mask)
}

func ipToUint32(ip string) (uint32, error) {
	octets := strings.Split(ip, ".")
	if len(octets) != 4 {
		return 0, errors.New("invalid IP address")
	}

	var io uint32

	o1, err := strconv.Atoi(octets[0])
	if err != nil {
		return 0, fmt.Errorf("failed to parse IP address, %s: %w", ip, err)
	}
	io += uint32(o1 << 24)
	o2, err := strconv.Atoi(octets[1])
	if err != nil {
		return 0, fmt.Errorf("failed to parse IP address, %s: %w", ip, err)
	}
	io += uint32(o2 << 16)
	o3, err := strconv.Atoi(octets[2])
	if err != nil {
		return 0, fmt.Errorf("failed to parse IP address, %s: %w", ip, err)
	}
	io += uint32(o3 << 8)
	o4, err := strconv.Atoi(octets[3])
	if err != nil {
		return 0, fmt.Errorf("failed to parse IP address, %s: %w", ip, err)
	}
	io += uint32(o4)

	return io, nil
}

func GetWSLIP(name string) (string, error) {
	ri, err := getRouteInfo(name)
	if err != nil {
		return "", err
	}

	cmd := exec.Command("wsl.exe", "-d", name, "--", "cat", "/proc/net/fib_trie")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	sout := string(out)
	sout = strings.TrimSpace(sout)
	if sout == "" {
		return "", errors.New("invalid output from fib_trie")
	}
	lines := strings.Split(sout, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		if strings.Index(line, "32 host LOCAL") != -1 {
			fs := strings.Fields(lines[i-1])
			ipstr := strings.TrimSpace(fs[1])
			ip, err := ipToUint32(ipstr)
			if err != nil {
				return "", fmt.Errorf("failed to convert ip, %s: %w", ipstr, err)
			}
			if isIPInRange(ri, ip) {
				return ipstr, nil
			}
		}
	}
	return "", errors.New("unable to find IP")
}

func initwsl() {
	cmd := exec.Command("wsl.exe", "-l")
	out, err := cmd.Output()
	if err != nil {
		slog.Error(err.Error())
		return
	}
	sout := UTF16toUTF8(out)
	sout = strings.TrimSpace(sout)
	if sout == "" {
		return
	}
	ws := strings.Split(sout, "\n")
	for i := 1; i < len(ws); i++ {
		items := strings.SplitN(strings.TrimSpace(ws[i]), " ", 2)
		name := strings.TrimSpace(items[0])
		ip, err := GetWSLIP(name)
		if err == nil {
			g_Tasks = append(g_Tasks, TaskConf{
				Name: name,
				Host: ip,
				Ports: []Port{
					{From: 5510, To: 5510},
					{From: 5710, To: 5710},
					{From: 5910, To: 5910},
				},
			})
		}
	}

	g_Tasks = append(g_Tasks,
		TaskConf{Name: "fedora", Host: "192.168.100.128", Ports: []Port{
			{From: 5510, To: 5510},
			{From: 5710, To: 5710},
			{From: 5910, To: 5910},
		}},
		TaskConf{Name: "ubuntu", Host: "192.168.100.34", Ports: []Port{
			{From: 5510, To: 5510},
			{From: 5710, To: 5710},
			{From: 5910, To: 5910},
		}},
	)
}
func UTF16toUTF8(input []byte) string {
	i := input[:len(input)/2]
	if len(i) == 0 {
		return ""
	}
	buf := (*[]uint16)(unsafe.Pointer(&i))

	out := utf16.Decode(*buf)
	return string(out)
}
