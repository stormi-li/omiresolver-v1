package main

import (
	"fmt"
	"net/url"

	"github.com/go-redis/redis/v8"
	"github.com/stormi-li/omiresolver-v1"
	"github.com/stormi-li/omiserd-v1"
)

var redisAddr = "118.25.196.166:3934"
var password = "12982397StrongPassw0rd"

func main() {
	resolver := omiresolver.NewResolver(&redis.Options{Addr: redisAddr, Password: password}, omiserd.Web)
	url, err := resolver.Resolve(url.URL{
		Host: "",
		Path: "/hello_server",
	})
	fmt.Println(err)
	fmt.Println(url.String())
}
