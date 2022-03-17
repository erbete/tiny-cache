package main

import (
	"fmt"
	"log"
	"tinycache"
)

func main() {
	cache, err := tinycache.NewCache(5, "5m")
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(cache)
}
