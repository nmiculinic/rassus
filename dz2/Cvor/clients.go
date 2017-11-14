package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"sync/atomic"
	"time"
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

type PackageType int

const (
	ACK = iota
	REQ
)

type Packet struct {
	Id      int64       `json:"id"`
	Type    PackageType `json:"type"`
	Payload []byte      `json:"payload"`
}

type ShittyConn struct {
	net.UDPConn
	avgDelay float64
	lossRate float64
}

func (conn *ShittyConn) WriteToUDP(b []byte, addr *net.UDPAddr) (int, error) {
	if rand.Float64() < conn.lossRate {
		log.Println("Fake dropping package", string(b[:20])+"...", "to", addr)
		return len(b), nil
	}
	delay := time.Duration(rand.ExpFloat64()*1000*conn.avgDelay) * time.Millisecond
	log.Println("Delay for package ", string(b[:20])+"...", "is ", delay)
	time.Sleep(delay)
	return conn.UDPConn.WriteToUDP(b, addr)
}

func handleConn(in, out, inUDPPackages chan []byte, conn *ShittyConn, addr *net.UDPAddr) {
	toAck := make(map[int64][]byte)
	var id int64 = 0
	ticker := time.Tick(1500 * time.Millisecond)
	for {
		select {
		case packet := <-inUDPPackages:
			var p Packet
			if err := json.Unmarshal(packet, &p); err != nil {
				log.Println(string(packet), err)
			} else {
				switch p.Type {
				case ACK:
					log.Println(addr, "reciever: Got ACK for", p.Id)
					delete(toAck, p.Id)
				case REQ:
					log.Println(addr, "reciever: Got REQ for", p.Id)
					if out, err := json.Marshal(Packet{
						Type: ACK,
						Id:   p.Id,
					}); err != nil {
						log.Println(err)
					} else {
						go func() {
							if _, err := conn.WriteToUDP(out, addr); err != nil {
								log.Println(err)
							}
						}()
					}
					in <- p.Payload
				default:
					log.Println("Unknown type", p.Type)
				}
			}
		case packet := <-out:
			pid := atomic.AddInt64(&id, 1)
			if out, err := json.Marshal(Packet{
				Payload: packet,
				Type:    REQ,
				Id:      pid,
			}); err != nil {
				log.Println(err)
			} else {
				toAck[pid] = out
				go func() {
					if _, err := conn.WriteToUDP(out, addr); err != nil {
						log.Println(err)
					} else {
						log.Println("Sent package with PID ", pid)
					}
				}()
			}
		case <-ticker:
			keys := make([]int64, 0, len(toAck))
			for k := range toAck {
				keys = append(keys, k)
			}
			log.Println("Missing", len(toAck), "acks for packages", keys, "for", addr)

			for pid, out := range toAck {
				go func(pid int64, out []byte) {
					if _, err := conn.WriteToUDP(out, addr); err != nil {
						log.Println(err)
					} else {
						log.Println("Resent package with PID ", pid)
					}
				}(pid, out)
			}
		}
	}
}

func UDProuter(conn *net.UDPConn, routes map[string]chan []byte) {
	buffer := make([]byte, 2048)
	for {
		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Println(err)
		} else {
			log.Println("Router Received packate from", addr)
			hop, ok := routes[fmt.Sprint(addr)]
			if !ok {
				log.Println(addr, "missing from routes", routes)
			} else {
				hop <- buffer[:n]
			}
		}
	}
}
