package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/google/uuid"
	"log"
	"math/rand"
	"net"
	"sync/atomic"
)

type Desc struct {
	Username string  `json:"username"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	IP       string  `json:"ip"`
	Port     int     `json:"port"`
}

func genDesc() (*Desc, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	return &Desc{
		Username: uuid.New().String(),
		Lon:      rand.Float64()*0.1 + 45.75,
		Lat:      rand.Float64()*0.13 + 15.87,
		IP:       conn.LocalAddr().(*net.UDPAddr).IP.String(),
	}, nil
}

type resp struct {
	Jsonrpc string      `json:"jsonrpc"`
	Error   interface{} `json:"error"`
	Result  interface{} `json:"result"`
	Id      int         `json:"id"`
}

type ServerConn struct {
	conn *bufio.ReadWriter
	id   int32
}

func (server *ServerConn) jsonrpc(method string, params interface{}) (interface{}, error) {
	id := atomic.LoadInt32(&server.id)
	atomic.AddInt32(&server.id, 1)
	req, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      id,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("sending", string(req))
	if _, err := server.conn.Write(append(req, []byte("\n")...)); err != nil {
		log.Fatal(err)
	}
	if err = server.conn.Flush(); err != nil {
		log.Fatal(err)
	}

	response, err := server.conn.ReadBytes('\n')
	if err != nil {
		log.Fatal(err)
	}

	log.Println("getting", string(response))
	r := resp{}
	err = json.Unmarshal(response, &r)
	if err != nil {
		log.Fatal("json decoding response", err)
	}
	if r.Error != nil {
		return nil, errors.New(fmt.Sprint(r.Error))
	}
	return r.Result, nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	ServerHost := *flag.String("ip", "localhost", "server hostname")
	serverPort := fmt.Sprint(*flag.Int("port", 3333, "server port"))
	server, err := net.ResolveTCPAddr("tcp", ServerHost+":"+serverPort)
	if err != nil {
		log.Fatal(err)
	}

	desc, err := genDesc()
	if err != nil {
		log.Fatal(err)
	}

	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	desc.Port = ln.Addr().(*net.TCPAddr).Port

	fmt.Printf("Server at [%s]; Me:%s\n", server, desc)

	conn, err := net.Dial("tcp", server.String())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	srv := &ServerConn{
		conn: bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)),
		id:   1,
	}

	resp, err := srv.jsonrpc("test", map[string]string{"username": desc.Username})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp)

	resp, err = srv.jsonrpc("register", desc)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp)
}
