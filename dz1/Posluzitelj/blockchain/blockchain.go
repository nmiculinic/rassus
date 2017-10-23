package blockchain

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

type block struct {
	Last     *block
	Username string
	Param    string
	Value    float64
	Id       int
	Hash     []byte
}

func (blk *block) Append(username string, parameter string, value float64) (*block, error) {
	sol := &block{
		Last:     blk,
		Username: username,
		Param:    parameter,
		Value:    value,
		Id:       blk.Id + 1,
	}
	h := sha256.New()
	h.Write([]byte(sol.Username))
	h.Write([]byte(sol.Param))
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], math.Float64bits(sol.Value))
	h.Write(buf[:])
	if sol.Last != nil {
		h.Write(sol.Last.Hash)
	}
	sol.Hash = h.Sum(nil)
	return sol, nil
}

func (blk *block) GetBlock(i int) (*block, error) {
	if blk.Id < i || i < 0 {
		return nil, errors.New(fmt.Sprint("No such Id!", i))
	}
	for blk.Id != i {
		blk = blk.Last
	}
	return blk, nil
}

func (blk *block) GetState() (map[string]map[string]float64, error) {
	sol := make(map[string]map[string]float64)
	curr := blk
	for curr != nil && curr.Last != nil {
		val, ok := sol[curr.Username]
		if !ok {
			sol[curr.Username] = make(map[string]float64)
			val = sol[curr.Username]
		}
		if _, ok := val[curr.Param]; !ok {
			val[curr.Param] = curr.Value
		}
		curr = curr.Last
	}
	return sol, nil
}

func New() *block {
	return &block{Id: 0}
}
