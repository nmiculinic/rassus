package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/google/uuid"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type Context struct {
	Username    string  `json:"username"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	IP          string  `json:"ip"`
	Port        int     `json:"port"`
	data        map[string]float64
	neighbour   *net.Conn
	neighReader *bufio.Reader
	lock        sync.Mutex
}

func genDesc() (*Context, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	return &Context{
		Username: uuid.New().String(),
		Lon:      rand.Float64()*0.1 + 45.75,
		Lat:      rand.Float64()*0.13 + 15.87,
		IP:       conn.LocalAddr().(*net.UDPAddr).IP.String(),
		data:     make(map[string]float64),
	}, nil
}

type resp struct {
	Jsonrpc string      `json:"jsonrpc"`
	Error   interface{} `json:"error"`
	Result  interface{} `json:"result"`
	Id      int         `json:"id"`
}

type ServerConn struct {
	conn_str *net.TCPAddr
	id       int32
}

func (server *ServerConn) jsonrpc(method string, params interface{}) (interface{}, error) {
	connRaw, err := net.Dial("tcp", server.conn_str.String())
	if err != nil {
		log.Panic(err)
	}
	defer connRaw.Close()
	conn := bufio.NewReadWriter(bufio.NewReader(connRaw), bufio.NewWriter(connRaw))

	id := atomic.LoadInt32(&server.id)
	atomic.AddInt32(&server.id, 1)
	req, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      id,
	})
	if err != nil {
		log.Panic(err)
	}
	log.Println("sending", string(req))
	if _, err := conn.Write(append(req, []byte("\n")...)); err != nil {
		log.Panic(err)
	}
	if err = conn.Flush(); err != nil {
		log.Panic(err)
	}

	response, err := conn.ReadBytes('\n')
	if err != nil {
		log.Panic(err)
	}

	log.Print("getting", string(response))
	r := resp{}
	err = json.Unmarshal(response, &r)
	if err != nil {
		log.Panic("json decoding response", err)
	}
	if r.Error != nil {
		return nil, errors.New(fmt.Sprint(r.Error))
	}
	return r.Result, nil
}

func gen_csv(csvFile string) (records [][]string, err error) {
	f, err := os.Open(csvFile)
	if err != nil {
		log.Panic(err)
		return
	}
	r := csv.NewReader(f)
	records, err = r.ReadAll()
	return
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	ServerStr := flag.String("srv", "localhost:3333", "server hostname")
	csvFile := flag.String("csv", "mjerenja.csv", "Location of csv file")
	flag.Parse()

	rec, err := gen_csv(*csvFile)
	server, err := net.ResolveTCPAddr("tcp", *ServerStr)
	if err != nil {
		log.Panic(err)
	}
	srv := &ServerConn{
		conn_str: server,
		id:       1,
	}

	ctx, err := genDesc()
	if err != nil {
		log.Panic(err)
	}

	if ln, err := net.Listen("tcp", ":0"); err != nil {
		log.Panic(err)
	} else {
		go handleSrv(ln, ctx)
		ctx.Port = ln.Addr().(*net.TCPAddr).Port
		log.Printf("Server at [%s]; Me:%s\n", server, ctx)
	}

	if resp, err := srv.jsonrpc("register", ctx); err != nil {
		log.Panic(err)
	} else {
		log.Println(resp)
	}
	testRPC(srv, ctx)

	getNeighbour(srv, ctx)

	startTime := time.Now()
	for {
		readMeasurement(startTime, rec, ctx, srv)
		time.Sleep(2500 * time.Millisecond)
	}
}

func fetchNeighbourMeasurement(ctx *Context, srv *ServerConn) (map[string]float64, error) {
	if ctx.neighbour == nil {
		addr, err := getNeighbour(srv, ctx)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		if conn, err := net.Dial("tcp", addr.String()); err != nil {
			log.Println(err)
			return nil, err
		} else {
			log.Println("Connected to client ", conn.RemoteAddr())
			ctx.neighbour = &conn
			ctx.neighReader = bufio.NewReader(conn)
		}
	}

	conn := *ctx.neighbour
	if _, err := conn.Write([]byte("Beam me up Spock!\n")); err != nil {
		log.Println(conn.RemoteAddr(), err)
		conn.Close()
		ctx.neighbour = nil
		ctx.neighReader = nil
		return nil, err
	}

	if data, err := ctx.neighReader.ReadBytes('\n'); err != nil {
		log.Println(err)
		return nil, err
	} else {
		sol := make(map[string]float64)
		err = json.Unmarshal(data, &sol)
		return sol, err
	}
}

func readMeasurement(startTime time.Time, rec [][]string, ctx *Context, srv *ServerConn) {
	elapsedSeconds := time.Now().Sub(startTime).Seconds()
	no := (int(elapsedSeconds) % 100) + 2
	log.Printf("Elapsed seconds %f, field %d, data %s\n", elapsedSeconds, no, rec[no])
	data := func() map[string]float64 {
		ctx.lock.Lock()
		defer ctx.lock.Unlock()
		for i := 0; i < len(rec[0]); i++ {
			param := rec[0][i]
			val, err := strconv.ParseFloat(rec[no][i], 64)
			if err != nil {
				log.Println(
					fmt.Sprintf(
						"Missing value for Row %d, param %s ",
						no,
						param), err)
				continue
			}
			ctx.data[param] = val
		}

		data := make(map[string]float64)
		for key, value := range ctx.data {
			data[key] = value
		}
		return data
	}()

	if neighbour, err := fetchNeighbourMeasurement(ctx, srv); err != nil {
		log.Println(err)
	} else {
		log.Println("Neighbour data: 	", neighbour)
		for key, nVal := range neighbour {
			if lVal, ok := data[key]; ok {
				data[key] = (lVal + nVal) / 2
				log.Println("Updating ", key, lVal, nVal)
			}
		}
	}

	for param, val := range data {
		if resp, err := srv.jsonrpc("storeMeasurement", map[string]interface{}{
			"username":     ctx.Username,
			"param":        param,
			"averageValue": val,
		}); err != nil {
			log.Panic(err)
		} else {
			log.Println("Got", resp)
		}
	}
}
func getNeighbour(srv *ServerConn, desc *Context) (*net.TCPAddr, error) {
	if resp, err := srv.jsonrpc("search", map[string]string{"username": desc.Username}); err != nil {
		log.Panic(err)
		return nil, err
	} else {
		log.Println("Nearest server: ", resp)
		if resp != nil {
			rr := resp.(map[string]interface{})
			if addr, err := net.ResolveTCPAddr("tcp", rr["ip"].(string)+":"+fmt.Sprint(rr["port"])); err != nil {
				log.Panic(err)
				return nil, err
			} else {
				log.Println("Found a mate :)!", rr["username"], addr)
				return addr, nil
			}
		} else {
			log.Println("I'm the lonely client...")
			return nil, nil
		}
	}
}

func testRPC(srv *ServerConn, desc *Context) {
	if resp, err := srv.jsonrpc("test", map[string]string{"username": desc.Username}); err != nil {
		log.Panic(err)
	} else {
		log.Println(resp)
	}
}

func handleSrv(ln net.Listener, ctx *Context) {
	defer ln.Close()
	for {
		if conn, err := ln.Accept(); err != nil {
			log.Println(err)
		} else {
			go handleConn(conn, ctx)
		}
	}
}
func handleConn(conn net.Conn, ctx *Context) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	singleRequest := func(req string) error {
		data, err := func() ([]byte, error) {
			ctx.lock.Lock()
			defer ctx.lock.Unlock()
			return json.Marshal(ctx.data)
		}()

		if err != nil {
			log.Println(err)
			return err
		}
		_, err = conn.Write(append(data, []byte("\n")...))
		return err
	}

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
		if err := singleRequest(string(recv)); err != nil {
			log.Println(err)
			return
		}
	}
}
