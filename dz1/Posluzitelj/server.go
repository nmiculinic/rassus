package main

import (
	"github.com/emicklei/go-restful"
	"github.com/nmiculinic/rassus/dz1/Posluzitelj/service"
	"log"
	"net/http"
)
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	restful.Add(service.New())
	log.Fatal(http.ListenAndServe(":3333", nil))
}
