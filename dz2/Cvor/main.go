package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/nmiculinic/rassus/dz2/Cvor/Data"
	"log"
	"math/rand"
	"net"
	"sort"
	"time"
)

type Message struct {
	ScalarTimestamp ScalarTimestamp `json:"scalar_timestamp"`
	VectorTimestamp VectorTimestamp `json:"vector_timestamp"`
	CO              float64         `json:"co"`
}

type Messages []*Message

func (s Messages) Len() int      { return len(s) }
func (s Messages) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type ByScalarMessage struct{ Messages }
type ByVectorMessage struct{ Messages }

func (s ByScalarMessage) Less(i, j int) bool {
	return s.Messages[i].ScalarTimestamp.Time.Before(s.Messages[j].ScalarTimestamp.Time)
}
func (s ByVectorMessage) Less(i, j int) bool {
	for k := range s.Messages[i].VectorTimestamp.Time {
		if s.Messages[i].VectorTimestamp.Time[k] > s.Messages[j].VectorTimestamp.Time[k] {
			return false
		}
	}
	return true
}

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

	strClients := make([]string, len(clients))
	for i, c := range clients {
		strClients[i] = fmt.Sprint(c)
	}
	vectorTimestamp := NewVectorTimestamp(strClients)
	scalarTimestamp := NewScalar()

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
	routerInwards := make(map[string]chan []byte)

	for _, client := range clients {
		routerInwards[fmt.Sprint(client)] = make(chan []byte)
		inward[fmt.Sprint(client)] = make(chan []byte)
		outward[fmt.Sprint(client)] = make(chan []byte)
	}

	go UDProuter(conn, routerInwards)
	for k := range inward {
		addr, err :=
			net.ResolveUDPAddr("udp", k)
		if err != nil {
			log.Panic(err)
		}

		go handleConn(
			inward[k],
			outward[k],
			routerInwards[k],
			&ShittyConn{
				UDPConn:  *conn,
				avgDelay: *avgDelay,
				lossRate: *lossRate,
			},
			addr,
		)
	}

	sending := time.Tick(1000 * time.Millisecond)
	processBatchTime := time.Tick(5 * time.Second)
	startTime := time.Now()
	messages := make([]*Message, 0)
	for {
		select {
		case <-sending:
			target := clients[rand.Int31n(int32(len(clients)))]
			log.Println("Sending to ", target)

			val, err := Data.ReadMeasurement(startTime, rec)
			if err != nil {
				log.Println(err)
				continue
			}
			vectorTimestamp = vectorTimestamp.Now()
			scalarTimestamp = scalarTimestamp.Now()
			m := Message{
				scalarTimestamp,
				vectorTimestamp,
				val,
			}
			messages = append(messages, &m)
			if b, err := json.Marshal(m); err != nil {
				log.Println(err)
			} else {
				log.Println("sending", string(b))
				log.Println("sending", vectorTimestamp)
				outward[fmt.Sprint(target)] <- b
			}
		case <-processBatchTime:
			var sum float64
			for _, el := range messages {
				sum += el.CO
			}
			avg := sum / float64(len(messages))
			log.Println("Average measurement", avg)

			sort.Sort(ByScalarMessage{messages})
			for _, el := range messages {
				log.Println(el.ScalarTimestamp.Time, el.CO)
			}

			sort.Sort(ByVectorMessage{messages})
			for _, el := range messages {
				log.Println(el.VectorTimestamp.Time, el.CO)
			}

			messages = make([]*Message, 0)
		default:
			for _, inChan := range inward {
				select {
				case recv := <-inChan:
					log.Println("Got from inward", string(recv))
					msg := &Message{}
					if err := json.Unmarshal(recv, msg); err != nil {
						log.Println(err)
					} else {
						scalarTimestamp = scalarTimestamp.Update(&msg.ScalarTimestamp, fmt.Sprint(me))
						vectorTimestamp = vectorTimestamp.Update(&msg.VectorTimestamp, fmt.Sprint(me))
						messages = append(messages, msg)
					}
				default:
				}
			}
		}
	}
}
