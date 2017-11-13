package main

import (
	"flag"
	"fmt"
	"github.com/nmiculinic/rassus/dz2/Cvor/Data"
	"log"
	"math/rand"
	"net"
	"time"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	csvFile := flag.String("csv", "mjerenja.csv", "Location of csv file")
	clientsFile := flag.String("clients", "clients.csv", "Location of clients file")
	id := flag.Int("id", -1, "Client id in clients file")
	lossRate := flag.Float64("avgLoss", 0.1, "Average loss rate")
	avgDelay := flag.Float64("avgDelay", 1.0, "Average delay rate in seconds")
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
	clients = append(clients[:*id], clients[*id+1:]...)
	log.Println("me", me)
	log.Println("Clients", clients)

	conn, err := net.ListenUDP("udp", me)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	inward := make(map[string]chan []byte)
	outward := make(map[string]chan []byte)
	router_inward := make(map[string]chan []byte)

	for _, client := range clients {
		router_inward[fmt.Sprint(client)] = make(chan []byte)
		inward[fmt.Sprint(client)] = make(chan []byte)
		outward[fmt.Sprint(client)] = make(chan []byte)
	}

	go UDProuter(conn, router_inward)
	for k := range inward {
		addr, err :=
			net.ResolveUDPAddr("udp", k)
		if err != nil {
			log.Panic(err)
		}

		go handleConn(
			inward[k],
			outward[k],
			router_inward[k],
			&ShittyConn{
				UDPConn: *conn,
				avgDelay: *avgDelay,
				lossRate: *lossRate,
				},
			addr,
		)
	}

	sending := time.Tick(2000 * time.Millisecond)

	for {
		select {
		case <-sending:
			log.Println("ovdje sam ")
			target := clients[rand.Int31n(int32(len(clients)))]
			log.Println("Sending to ", target)
			outward[fmt.Sprint(target)] <- []byte("Ok sam")
		default:
			target := clients[rand.Int31n(int32(len(clients)))]
			select {
			case recv := <-inward[fmt.Sprint(target)]:
				log.Println("Got from inward", string(recv))
			default:
			}
		}
	}
}
