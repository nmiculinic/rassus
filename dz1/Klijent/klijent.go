package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/google/uuid"
	"gopkg.in/resty.v1"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
	"github.com/nmiculinic/rassus/dz1/interfaces"
	"net/http"
)

type Context struct {
	desc        interfaces.Vertex
	data        map[string]float64
	neighbour   *net.Conn
	neighReader *bufio.Reader
	lock        sync.Mutex
	connStr     string
}

func genDesc() (*Context, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	return &Context{
		desc: interfaces.Vertex{
			Username: uuid.New().String(),
			Lon:      rand.Float64()*0.1 + 45.75,
			Lat:      rand.Float64()*0.13 + 15.87,
			Ip:       conn.LocalAddr().(*net.UDPAddr).IP,
		},
		data:     make(map[string]float64),
	}, nil
}

func (ctx *Context) register() error {
	if ctx.connStr == "" {
		return errors.New("No connection string for server")
	}

	if resp, err := resty.R().
		SetBody(ctx.desc).
		Post(fmt.Sprintf("http://%s/register", ctx.connStr)); err != nil {
		log.Println(resp, resp.StatusCode())
		if resp.StatusCode() != http.StatusOK {
			return errors.New(fmt.Sprint(resp, resp.StatusCode()))
		} else {
			return nil
		}
	} else {
		return err
	}
}

func gen_csv(csvFile string) (records [][]string, err error) {
	f, err := os.Open(csvFile)
	defer f.Close()
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

	ctx, err := genDesc()
	if err != nil {
		log.Panic(err)
	}
	ctx.connStr = *ServerStr

	rec, err := gen_csv(*csvFile)

	if err := ctx.register(); err != nil {
		log.Panic(err)
	}

	if ln, err := net.Listen("tcp", ":0"); err != nil {
		log.Panic(err)
	} else {
		go handleSrv(ln, ctx)
		ctx.desc.Port = ln.Addr().(*net.TCPAddr).Port
		log.Printf("Server at [%s]; Me:%s\n", ctx.connStr, ctx)
	}


	ctx.getNeighbour()

	startTime := time.Now()
	for {
		if data, err := ctx.readMeasurement(startTime, rec); err != nil {
			log.Println(err)
		} else {
			log.Println(data)
			for param, val := range data {
				if err := ctx.storeMeasurement(param, val); err != nil {
					log.Panic("Cannot store measurement!", err)
				}
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (ctx *Context) fetchNeighbourMeasurement() (map[string]float64, error) {
	if ctx.neighbour == nil {
		addr, err := ctx.getNeighbour()
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

func (ctx *Context) readMeasurement(startTime time.Time, rec [][]string) (map[string]float64, error) {
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

	if neighbour, err := ctx.fetchNeighbourMeasurement(); err != nil {
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
	return data, nil

}
func (ctx *Context) getNeighbour() (*net.TCPAddr, error) {
	if resp, err := resty.R().
		SetHeader("Accept", "application/json").
		Get(fmt.Sprintf("http://%s/search/%s", ctx.connStr, ctx.desc.Username)); err != nil {
		log.Println(err)
		return nil, err
	} else {
		log.Println("Nearest server: ", resp, resp.StatusCode())
		if resp.StatusCode() != http.StatusOK {
			return nil, errors.New(fmt.Sprint(resp))
		}
		var rr interfaces.Vertex
		json.Unmarshal(resp.Body(), &rr)
		if addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", rr.Ip, rr.Port)); err != nil {
			log.Panic(err)
			return nil, err
		} else {
			log.Println("Found a mate :)!", rr.Username, addr)
			return addr, nil
		}
	}
}

func (ctx *Context) storeMeasurement(param string, value float64) error {
	m := interfaces.Measurement{
		Username:ctx.desc.Username,
		Param:param,
		Value:value,
	}
	if resp, err := resty.R().
		SetBody(m).
		Post(fmt.Sprintf("http://%s/storeMeasurement", ctx.connStr)); err != nil {
		log.Println(resp)
		if resp.StatusCode() != http.StatusOK {
			return errors.New(fmt.Sprint(resp))
		}
		return nil
	} else {
		return err
	}
}

func handleSrv(ln net.Listener, ctx *Context) {
	defer ln.Close()

	handleConn := func(conn net.Conn) {
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

	for {
		if conn, err := ln.Accept(); err != nil {
			log.Println(err)
		} else {
			go handleConn(conn)
		}
	}
}
