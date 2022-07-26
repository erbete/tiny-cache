package main

import (
	"log"
	"tinycache"
)

func main() {
	cache, err := tinycache.NewCache(5, "5m")
	if err != nil {
		log.Fatalln(err)
	}

	_ = cache
}
