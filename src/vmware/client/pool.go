package client

import (
	"errors"
	"log"
	"main/models"
	"time"
)

type PoolItem struct {
	Node Client
	buzy bool
}

type Pool []*PoolItem

func NewPool(conf models.Conf, capacity int) (Pool, error) {
	pool := Pool{}
	for i := 0; i < capacity; i++ {
		c, err := getClient(conf)
		if err != nil {
			return Pool{}, err
		}
		pool = append(pool, &PoolItem{Node: c, buzy: false})
	}
	return pool, nil
}

func (p Pool) GetClient(timeout time.Duration) (*PoolItem, error) {
	now := time.Now()
	for {
		for i := range p {
			if !p[i].buzy {
				p[i].buzy = true
				log.Println("Клиент выдан из пула")
				return p[i], nil
			}
		}
		if time.Since(now) > timeout {
			break
		}
	}
	return &PoolItem{}, errors.New("VMWare connection pool timeout excided.")
}

func (p Pool) PutClient(client *PoolItem) {
	client.buzy = false
	client.Node.Ctx.Deadline()
	log.Println("Клиент возвращен в пул")
}
