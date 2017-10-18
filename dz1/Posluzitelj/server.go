package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"os"
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
//String IPaddress, int port)), druga metoda vraća informacije o geografski
//najbližem senzoru među senzorima koji su trenutno spojeni na poslužitelj (
//UserAddress search Neighbour(String username)), a treća metoda služi
//za prijavljivanje umjerenih podataka ( boolean storeMeasurement(String
//username, String parameter, float averageValue) ). Neka se prilikom
//prijavljivanja umjerenih podataka ažurira i stanje blok-lanca. U

type Vertex struct {
	lat  float64
	lon  float64
	ip   net.IP
	port int
}

func square(x float64) float64 {
	return x * x
}

func (this *Vertex) dist(other *Vertex) float64 {
	if this == nil {
		return math.Inf(+1)
	}

	R := 6371.0 // Earth diameter
	dlon := other.lon - this.lon
	dlat := other.lat - this.lat
	a := square(math.Sin(dlat/2)) + math.Cos(this.lat)*math.Cos(other.lat)*square(math.Sin(dlon/2))
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
	if _, ok := state.sensors[username]; !ok {
		return false, errors.New("Sensor already exists")
	}
	state.sensors[username] = &Vertex{
		lat:  lat,
		lon:  lon,
		ip:   net.ParseIP(ip),
		port: port,
	}
	return true, nil
}

func (state *SensorState) search(username string) (*Vertex, error) {
	state.mutex.Lock()
	defer state.mutex.Unlock()
	if _, ok := state.sensors[username]; !ok {
		return nil, errors.New(fmt.Sprintf("Cannot found %s in sensors list", username))
	}
	var sol *Vertex = nil
	var dist float64 = math.Inf(+1)
	for k, v := range state.sensors {
		if k != username {
			if sol.dist(v) < dist {
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
	return false, errors.New("Not yet implemented")
}

func main() {
	state := &SensorState{
		sensors: make(map[string]*Vertex),
	}

	// Listen for incoming connections.
	l, err := net.Listen(CONN_TYPE, CONN_HOST+":"+CONN_PORT)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	// Close the listener when the application closes.
	defer l.Close()
	fmt.Println("Listening on " + CONN_HOST + ":" + CONN_PORT)
	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		// Handle connections in a new goroutine.
		go handleRequest(state, conn)
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

func (req *request) handleResponse(sol interface{}, err error, conn net.Conn){
	if err != nil {
		conn.Write([]byte(fmt.Sprintf(
			`{"jsonrpc": "2.0", "error": {"code": -32000, "message": "Server error %s"}, "id": "%d"}`, err, req.Id)))
	} else {
		b, err := json.Marshal(sol)
		if err != nil {
			conn.Write([]byte(fmt.Sprintf(
				`{"jsonrpc": "2.0", "error": {"code": -32001, "message": "Server error %s"}, "id": "%d"}`, err, req.Id)))
		} else {
			conn.Write([]byte(fmt.Sprintf(
				`{"jsonrpc": "2.0", "result": %s, "id": "%d"}`, b, req.Id)))
		}
	}
}

func handleRequest(state *SensorState, conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	var err error
	recv, err := reader.ReadBytes('\n')
	if err != nil {
		fmt.Println("Error reading:", err.Error())
		conn.Close()
	}

	req := request{}
	err = json.Unmarshal(recv, &req)
	if err != nil {
		conn.Write([]byte(fmt.Sprintf(
			`{"jsonrpc": "2.0", "error": {"code": -32700, "message": "%s"}, "id": null}`,
			err,
		)))
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Errorf("Error happened %s", r)
			conn.Write([]byte(fmt.Sprintf(
				`{"jsonrpc": "2.0", "error": {"code": -32603, "message": "Internal error"}, "id": "%d"}`, req.Id)))
		}
	}()

	switch req.Method {
	case "test":
		sol, err := state.test(req.Params["username"].(string))
		req.handleResponse(sol, err, conn)
	case "request":
		sol, err := state.register(
			req.Params["username"].(string),
			req.Params["lat"].(float64),
			req.Params["lon"].(float64),
			req.Params["IP"].(string),
			req.Params["port"].(int),
		)
		req.handleResponse(sol, err, conn)
	case "search":
		sol, err := state.search(
			req.Params["username"].(string),
		)
		req.handleResponse(sol, err, conn)
	case "storeMeasurement":
		sol, err := state.storeMeasurement(
			req.Params["username"].(string),
			req.Params["param"].(string),
			req.Params["averageValue"].(float64),
		)
		req.handleResponse(sol, err, conn)
	default:
		conn.Write([]byte(fmt.Sprintf(
			`{"jsonrpc": "2.0", "error": {"code": -32601, "message": "Method not found"}, "id": "%d"}`, req.Id)))
	}
	fmt.Printf(`"%s" ID: %s`, req.Jsonrpc, req.Id)
	fmt.Println()
	fmt.Println(req)
}
