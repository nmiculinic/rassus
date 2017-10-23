package service

import (
	"errors"
	"github.com/emicklei/go-restful"
	"github.com/nmiculinic/rassus/dz1/Posluzitelj/blockchain"
	"github.com/nmiculinic/rassus/dz1/interfaces"
	"log"
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
	mutex := new(sync.Mutex)
	block := blockchain.New()
	blockMutex := sync.Mutex{}

	service.Route(service.POST("/register").To(func(request *restful.Request, response *restful.Response) {
		mutex.Lock()
		defer mutex.Unlock()
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
		}

		sensors[args.Username] = args
		if err := response.WriteEntity(map[string]bool{"result": true}); err != nil {
			log.Println(err)
		}
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

	return service
}
