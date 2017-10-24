package interfaces

import (
	"math"
	"net"
)

type Measurement struct {
	Username string  `json:"username"`
	Param    string  `json:"param"`
	Value    float64 `json:"value"`
}

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

func (this *Vertex) Dist(other *Vertex) float64 {
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
