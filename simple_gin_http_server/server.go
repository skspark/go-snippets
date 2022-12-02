package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

const (
	TidCtxKey    = "__tid__"
	TidHeaderKey = "RequestId"
)

func TidMiddleware(c *gin.Context) {
	var tid string
	if existingID, ok := c.Get(TidCtxKey); ok {
		tid = existingID.(string)
	} else if existingID := c.GetHeader(TidHeaderKey); existingID != "" {
		tid = existingID
	} else {
		tid = RandStringBytes(20)
	}
	c.Set(TidCtxKey, tid)
	c.Next()
	c.Header(TidHeaderKey, tid)
}

func PingRouter(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	c.String(http.StatusOK, "pong %s", name)
}

type HTTPServerConfig struct {
	Port int
}

type HTTPServer struct {
	config HTTPServerConfig
	engine *gin.Engine
}

func NewHTTPServer(
	ctx context.Context,
	config HTTPServerConfig,
) *HTTPServer {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(TidMiddleware)

	r.GET("/ping", PingRouter)
	return &HTTPServer{
		config: config,
		engine: r,
	}
}

func (s *HTTPServer) Start() error {
	return s.engine.Run(fmt.Sprintf(":%d", s.config.Port))
}

func main() {
	ctx := context.Background()
	conf := HTTPServerConfig{
		Port: 8080,
	}
	go func(serv *HTTPServer) {
		serv.Start()
	}(NewHTTPServer(ctx, conf))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}
