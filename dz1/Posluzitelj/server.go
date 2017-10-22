package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"runtime/debug"
	"sync"
)

const (
	CONN_HOST = "localhost"
	CONN_PORT = "3333"
	CONN_TYPE = "tcp"
)

//oslužitelja koji komuniciraju koristeći web-usluge. Poslužitelj treba implementirati
//3 metode. Jedna metoda će služiti za registraciju senzora kod poslužitelja ( boolean
//register(String username, double latitude, double longitude,
//String IPaddress, int Port)), druga metoda vraća informacije o geografski
//najbližem senzoru među senzorima koji su trenutno spojeni na poslužitelj (
//UserAddress search Neighbour(String username)), a treća metoda služi
//za prijavljivanje umjerenih podataka ( boolean storeMeasurement(String
//username, String parameter, float averageValue) ). Neka se prilikom
//prijavljivanja umjerenih podataka ažurira i stanje blok-lanca. U

type Vertex struct {
	Username string  `json:"username"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	Ip       net.IP  `json:"ip"`
	Port     int     `json:"port"`
}

func square(x float64) float64 {
	return x * x
}

func (this *Vertex) dist(other *Vertex) float64 {
	if this == nil {
		return math.Inf(+1)
	}

	R := 6371.0 // Earth diameter
	dlon := other.Lon - this.Lon
	dlat := other.Lat - this.Lat
	a := square(math.Sin(dlat/2)) + math.Cos(this.Lat)*math.Cos(other.Lat)*square(math.Sin(dlon/2))
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	d := R * c
	return d
}

type SensorState struct {
	sensors map[string]*Vertex
	mutex   sync.Mutex
}

func (state *SensorState) register(username string, lat, lon float64, ip string, port int) (bool, error) {
	state.mutex.Lock()
	defer state.mutex.Unlock()
	log.Println(state.sensors)
	if _, fail := state.sensors[username]; fail {
		log.Println(state.sensors)
		log.Println(fail)
		return false, errors.New("Sensor already exists")
	}
	state.sensors[username] = &Vertex{
		Username: username,
		Lat:      lat,
		Lon:      lon,
		Ip:       net.ParseIP(ip),
		Port:     port,
	}
	return true, nil
}

func (state *SensorState) search(username string) (*Vertex, error) {
	state.mutex.Lock()
	defer state.mutex.Unlock()
	target, ok := state.sensors[username]
	if !ok {
		return nil, errors.New(fmt.Sprintf("Cannot found %s in sensors list", username))
	}
	var sol *Vertex = nil
	var dist float64 = math.Inf(+1)
	for k, v := range state.sensors {
		log.Println(k, username, dist)
		if k != username {
			if sol == nil || target.dist(v) < dist {
				sol = v
				dist = sol.dist(v)
			}
		}
	}
	return sol, nil
}

func (state *SensorState) storeMeasurement(username string, parameter string, averageValue float64) (bool, error) {
	state.mutex.Lock()
	defer state.mutex.Unlock()
	log.Println("To be implemented!!!")
	return true, nil
	//return false, errors.New("Not yet implemented")
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	state := &SensorState{
		sensors: make(map[string]*Vertex),
	}

	// Listen for incoming connections.
	l, err := net.Listen(CONN_TYPE, CONN_HOST+":"+CONN_PORT)
	if err != nil {
		log.Fatal(err)
	}
	// Close the listener when the application closes.
	defer l.Close()
	fmt.Println("Listening on " + CONN_HOST + ":" + CONN_PORT)
	for {
		if conn, err := l.Accept(); err != nil {
			log.Fatal(err)
		} else {
			go handleRequest(state, conn)
		}
	}
}

type request struct {
	Jsonrpc string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params"`
	Id      int                    `json:"id"`
}

func (state *SensorState) test(username string) (string, error) {
	state.mutex.Lock()
	defer state.mutex.Unlock()
	return fmt.Sprintf("Username is %s", username), nil
}

func (req *request) handleResponse(sol interface{}, err error, conn net.Conn) {
	if err != nil {
		conn.Write([]byte(fmt.Sprintf(
			`{"jsonrpc": "2.0", "error": {"code": -32000, "message": "Server error %s"}, "id": %d}`+"\n", err, req.Id)))
	} else {
		b, err := json.Marshal(sol)
		if err != nil {
			log.Println(err)
			conn.Write([]byte(fmt.Sprintf(
				`{"jsonrpc": "2.0", "error": {"code": -32001, "message": "Server error %s"}, "id": %d}`+"\n", err, req.Id)))
		} else {
			conn.Write([]byte(fmt.Sprintf(
				`{"jsonrpc": "2.0", "result": %s, "id": %d}`+"\n", b, req.Id)))
		}
	}
}

func handleRequest(state *SensorState, conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	defer func() {
		if r := recover(); r != nil {
			log.Printf("%s: %s", r, debug.Stack()) // line 20
		}
	}()

	for {
		recv, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				log.Println(conn.RemoteAddr(), "Connection closed")
			} else {
				log.Println(err)
			}
			return
		}
		fmt.Println("got: ", string(recv))
		if err := singleRequest(recv, conn, state); err != nil {
			log.Println(err)
			return
		}
	}
}
func singleRequest(recv []byte, conn net.Conn, state *SensorState) error {
	req := request{}
	if err := json.Unmarshal(recv, &req); err != nil {
		conn.Write([]byte(fmt.Sprintf(
			`{"jsonrpc": "2.0", "error": {"code": -32700, "message": "%s"}, "id": null}`+"\n",
			err,
		)))
		log.Println(err)
		return err
	}

	switch req.Method {
	case "test":
		sol, err := state.test(req.Params["username"].(string))
		req.handleResponse(sol, err, conn)
	case "register":
		sol, err := state.register(
			req.Params["username"].(string),
			req.Params["lat"].(float64),
			req.Params["lon"].(float64),
			req.Params["ip"].(string),
			int(req.Params["port"].(float64)),
		)
		req.handleResponse(sol, err, conn)
	case "search":
		sol, err := state.search(
			req.Params["username"].(string),
		)
		if sol == nil {
			req.handleResponse(sol, err, conn)
		} else {
			log.Println(*sol)
			req.handleResponse(*sol, err, conn)
		}
	case "storeMeasurement":
		sol, err := state.storeMeasurement(
			req.Params["username"].(string),
			req.Params["param"].(string),
			req.Params["averageValue"].(float64),
		)
		req.handleResponse(sol, err, conn)
	default:
		conn.Write([]byte(fmt.Sprintf(
			`{"jsonrpc": "2.0", "error": {"code": -32601, "message": "Method not found"}, "id": "%d"}`+"\n", req.Id)))
	}
	return nil
}
