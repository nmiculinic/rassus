package service

import (
	"errors"
	"github.com/emicklei/go-restful"
	"github.com/nmiculinic/rassus/dz1/Posluzitelj/blockchain"
	"github.com/nmiculinic/rassus/dz1/interfaces"
	"log"
	"math"
	"net/http"
	"sync"
)

type User struct {
	Id, Name string
}

func New() *restful.WebService {
	service := new(restful.WebService)
	service.
		Path("/").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	sensors := make(map[string]*interfaces.Vertex)
	sensorMutex := new(sync.Mutex)
	block := blockchain.New()
	blockMutex := sync.Mutex{}

	service.Route(service.POST("/register").To(func(request *restful.Request, response *restful.Response) {
		sensorMutex.Lock()
		defer sensorMutex.Unlock()
		log.Println(sensors)
		args := &interfaces.Vertex{}
		if err := request.ReadEntity(args); err != nil {
			log.Panic(err)
		}
		log.Println(args)
		// here you would fetch user from some persistence system
		if _, fail := sensors[args.Username]; fail {
			log.Println(sensors)
			log.Println(fail)
			response.WriteError(http.StatusUnprocessableEntity, errors.New("Sensor already exists"))
		} else {
			sensors[args.Username] = args
			if err := response.WriteEntity(map[string]bool{"result": true}); err != nil {
				log.Println(err)
			}
		}
		log.Println("Known sensors:", sensors)
	}))
	service.Route(service.POST("/storeMeasurement").To(func(request *restful.Request, response *restful.Response) {
		blockMutex.Lock()
		defer blockMutex.Unlock()

		args := &interfaces.Measurement{}
		if err := request.ReadEntity(args); err != nil {
			log.Panic(err)
		}
		log.Println(args)

		if blk, err := block.Append(args.Username, args.Param, args.Value); err != nil {
			response.WriteError(http.StatusUnprocessableEntity, err)
		} else {
			if blk.Id%100 == 0 {
				state, _ := blk.GetState()
				log.Println("Blockchain current state:\n", state)
			}
			block = blk
			if err := response.WriteEntity(map[string]bool{"result": true}); err != nil {
				log.Println(err)
			}
		}
	}))
	service.Route(service.GET("/search/{username}").To(func(request *restful.Request, response *restful.Response) {
		sensorMutex.Lock()
		sensorMutex.Unlock()

		username := request.PathParameter("username")
		if target, ok := sensors[username]; !ok {
			response.WriteError(http.StatusPreconditionFailed, errors.New("No such username "+username))
		} else {
			var sol *interfaces.Vertex = nil
			var dist float64 = math.Inf(+1)
			for k, v := range sensors {
				if k != username {
					if sol == nil || target.Dist(v) < dist {
						sol = v
						dist = sol.Dist(v)
					}
				}
			}
			if sol != nil {
				response.WriteEntity(*sol)
			} else {
				response.WriteError(http.StatusNotFound, errors.New("No neighbours to return"))
			}
		}
	}))

	return service
}
