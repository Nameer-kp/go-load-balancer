package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/Nameer-kp/go-load-balancer/backend"
	"github.com/Nameer-kp/go-load-balancer/helpers"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

var serverpool ServerPool

func healthCheck() {
	t := time.NewTimer(time.Second * 20)
	for {
		select {
		case <-t.C:
			log.Println("Starting health check. ..")
			serverpool.HealthCheck()
			log.Println("Health check completed")
		}
	}
}

func main() {
	var serverList string
	var port int
	flag.StringVar(&serverList, "backends", "", "Load balanced backends, use commas to seperate")
	flag.IntVar(&port, "port", 3030, "Port to serve")
	flag.Parse()

	if len(serverList) == 0 {
		log.Fatal("Please provide atleast one backends")
	}
	tokens := strings.Split(serverList, ",")
	for _, tok := range tokens {
		serverUrl, err := url.Parse(tok)
		if err != nil {
			log.Fatal(err)
		}
		proxy := httputil.NewSingleHostReverseProxy(serverUrl)
		proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, err error) {
			log.Printf("[%s] %s\n", serverUrl.Host, err.Error())
			retries := helpers.GetRetryFromContext(request)
			if retries < 3 {
				select {
				case <-time.After(10 * time.Millisecond):
					ctx := context.WithValue(request.Context(), helpers.Retry, retries+1)
					proxy.ServeHTTP(writer, request.WithContext(ctx))
				}
				return
			}
			serverpool.MarkBackendStatus(serverUrl, false)
			attempts := helpers.GetAttemptsFromContext(request)
			log.Printf("%s(%s) Attempting retry %d\n", request.RemoteAddr, request.URL.Path, attempts)
			ctx := context.WithValue(request.Context(), helpers.Attempts, attempts+1)
			lb(writer, request.WithContext(ctx))
		}
		serverpool.AddBackend(&backend.Backend{
			URL:          serverUrl,
			Alive:        true,
			ReverseProxy: proxy,
		})
	}

	server := http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(lb),
	}
	go healthCheck()
	log.Printf("Load balancer started at :%d \n", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}

}
