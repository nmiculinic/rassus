package main

import (
	"log"
	"flag"
	"github.com/nmiculinic/rassus/dz2/Cvor/Data"
	"time"
	"fmt"
	"net"
	"math/rand"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	csvFile := flag.String("csv", "mjerenja.csv", "Location of csv file")
	clientsFile := flag.String("clients", "clients.csv", "Location of clients file")
	id := flag.Int("id", -1, "Client id in clients file")
	flag.Parse()

	rec, err := Data.ReadCSV(*csvFile)
	if err != nil {
		log.Panic(err)
	}
	log.Println(rec[0])

	clients, err := ReadClients(*clientsFile)
	if err != nil {
		log.Panic(err)
	}
	log.Println(clients)
	if *id < 0 || *id >= len(clients) {
		log.Panic("Invalid id, not in range!")
	}

	me := clients[*id]
	log.Println(me)

	conn, err := net.ListenUDP("udp", me)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()
	go udpListener(conn)

	for {
		select {
			case <- time.Tick(1 * time.Second):
				fmt.Println("Ticked!")
			case <- time.Tick(500 * time.Millisecond):
				target := me
				for ;target == me;{
					target = clients[rand.Int31n(int32(len(clients)))]
				}
				if _, err := conn.WriteToUDP([]byte("ok sam"), target); err != nil {
					log.Println(err)
				} else {
					log.Println("Sent package to ", target)
				}
		}
	}
}

func udpListener(conn *net.UDPConn) {
	buf := make([]byte, 1024)
	for {
		n,addr,err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Println("Error: ",err)
		} else {
			log.Println("Received ",string(buf[0:n]), " from ",addr)
			if string(buf[0:n]) != "Ack!" {
				respFn:= func() {
					//time.Sleep(1 * time.Second)
					conn.WriteToUDP([]byte("Ack!"), addr)
				}
				go respFn()
			}
		}
	}
}
