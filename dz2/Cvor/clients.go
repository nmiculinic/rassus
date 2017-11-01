package main

import (
	"os"
	"bufio"
	"net"
)

func ReadClients(csvFile string) ([]*net.UDPAddr, error) {
	f, err := os.Open(csvFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var sol []*net.UDPAddr
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if addr, err := net.ResolveUDPAddr("udp", line); err != nil {
			return nil, err
		} else {
			sol = append(sol, addr)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return sol, nil
}
