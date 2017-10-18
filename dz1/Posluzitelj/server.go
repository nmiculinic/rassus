package main

import (
	"errors"
	"fmt"
	"math"
	"net"
	"os"
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

	R := 6371.0  // Earth diameter
	dlon := other.lon - this.lon
	dlat := other.lat - this.lat
	a := square(math.Sin(dlat/2)) + math.Cos(this.lat)*math.Cos(other.lat)*square(math.Sin(dlon/2))
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	d := R * c
	return d
}

type SensorState struct {
	sensors map[string]*Vertex
}

func (state *SensorState) register(username string, lat, lon float64, ip string, port int) bool {
	if _, ok := state.sensors[username]; !ok {
		return false
	}
	state.sensors[username] = &Vertex{
		lat:  lat,
		lon:  lon,
		ip:   net.ParseIP(ip),
		port: port,
	}
	return true
}

func (state *SensorState) search(username string) (*Vertex, error) {
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

func (state *SensorState) storeMeasurement(username string, parameter string, averageValue float64)  bool {
	return false
}

func main() {
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
		go handleRequest(conn)
	}
}

// Handles incoming requests.
func handleRequest(conn net.Conn) {
	// Make a buffer to hold incoming data.
	buf := make([]byte, 1024)
	// Read the incoming connection into the buffer.
	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading:", err.Error())
	}
	// Send a response back to person contacting us.
	conn.Write([]byte("Message received."))
	// Close the connection when you're done with it.
	conn.Close()
}
