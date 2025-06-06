package httpbutler

import (
	"fmt"

	echo "github.com/labstack/echo/v4"
)

type EchoServer interface {
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PATCH(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	OPTIONS(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	HEAD(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
}

type EndpointParent interface {
	GetEcho() EchoServer
	GetMiddlewares() []Middleware
	GetPath() string
	GetAuthHandlers() []AuthHandler
}

type EndpointInterface interface {
	Register(server EndpointParent) []EndpointInterface
}

type Server struct {
	Port        int
	echo        *echo.Echo
	endpoints   []EndpointInterface
	middlewares []Middleware
}

func CreateServer() *Server {
	e := echo.New()
	return &Server{
		Port:      80,
		echo:      e,
		endpoints: []EndpointInterface{},
	}
}

func (server *Server) GetEcho() EchoServer {
	return server.echo
}

func (server *Server) GetMiddlewares() []Middleware {
	return server.middlewares
}

func (server *Server) GetPath() string {
	return ""
}

func (server *Server) GetAuthHandlers() []AuthHandler {
	return []AuthHandler{}
}

func (server *Server) Add(endpoint EndpointInterface) {
	routes := endpoint.Register(server)
	server.endpoints = append(server.endpoints, routes...)
}

func (server *Server) Use(middleware Middleware) {
	server.middlewares = append(server.middlewares, middleware)
}

func (server *Server) Listen() {
	server.echo.Start(fmt.Sprintf(":%v", server.Port))
}

func (server *Server) Close() {
	server.echo.Close()
}
