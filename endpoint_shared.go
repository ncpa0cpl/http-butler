package httpbutler

import (
	echo "github.com/labstack/echo/v4"
)

type AnyEndpoint interface {
	GetPath() string
	GetMethod() string
	GetAuth() AuthHandler
	GetEncoding() string
	ExecuteHandler(ctx echo.Context, request *Request) *Response
	GetCachePolicy() *HttpCachePolicy
	GetStreamingSettings() *StreamingSettings
	GetMiddlewares() []Middleware
}

func registerEndpoint[E AnyEndpoint](e E, parent EndpointParent) {
	echoServer := parent.GetEcho()
	basepath := parent.GetPath()
	middlewares := append(parent.GetMiddlewares(), e.GetMiddlewares()...)
	authHandlers := parent.GetAuthHandlers()
	defaultEncoding := e.GetEncoding()
	cachePolicy := e.GetCachePolicy()
	streamSettings := e.GetStreamingSettings()
	fullpath := pathJoin(basepath, e.GetPath())

	reqMiddlewares := getReqMiddlewares(middlewares)
	respMiddlewares := getRespMiddlewares(middlewares)

	endpAuth := e.GetAuth()
	if endpAuth != nil {
		authHandlers = append(authHandlers, endpAuth)
	}

	if defaultEncoding == "" {
		defaultEncoding = "auto"
	}

	handler := func(ctx echo.Context) error {
		request := NewRequest(ctx)

		for _, authHandler := range authHandlers {
			auth := authHandler(request)
			if !auth.IsSuccessful() {
				return auth.SendResponse(request)
			}
		}

		var response *Response
		for _, md := range reqMiddlewares {
			err := md.OnRequest(
				request,
				func(nextReq *Request) {
					request = nextReq
				},
				func(sendInstead *Response) {
					response = sendInstead
				},
			)

			if err != nil {
				ctx.Logger().Errorf("middleware %s request handler returned an error", md.Name)
				return err
			}

			if response != nil {
				break
			}
		}

		if response == nil {
			response = e.ExecuteHandler(ctx, request)
		}

		for _, md := range respMiddlewares {
			err := md.OnResponse(
				request,
				response,
				func(sendInstead *Response) {
					response = sendInstead
				},
			)

			if err != nil {
				ctx.Logger().Errorf("middleware %s response handler returned an error", md.Name)
				return err
			}
		}

		if response == nil {
			ctx.Logger().Errorf("endpoint handler did not return a response [path=%s]", fullpath)
			return ctx.NoContent(500)
		}

		if response.customHandler != nil {
			return response.send(request)
		}

		if response.StreamingSettings == nil {
			response.StreamingSettings = streamSettings
		}

		if response.Status < 300 {
			cp := resolveCachePolicy(cachePolicy, response)
			if cp != nil {
				response.Headers.Set("Cache-Control", cp.ToString())

				if !cp.DisableETagGeneration {
					AddEtag(response)
				}

				if !cp.DisableAutoResponseSkipping {
					etag := response.Headers.Get("ETag")
					if etag != "" {
						ifNoneMatch := request.Headers.Get("If-None-Match")
						if etag == ifNoneMatch {
							response.Headers.CopyInto(ctx.Response().Header())
							return ctx.NoContent(304)
						}
					}
				}
			}
		}

		if response.Encoding == "" {
			response.Encoding = defaultEncoding
		}

		return response.send(request)
	}

	switch e.GetMethod() {
	case "GET":
		echoServer.GET(fullpath, handler)
		return
	case "POST":
		echoServer.POST(fullpath, handler)
		return
	case "PUT":
		echoServer.PUT(fullpath, handler)
		return
	case "PATCH":
		echoServer.PATCH(fullpath, handler)
		return
	case "DELETE":
		echoServer.DELETE(fullpath, handler)
		return
	case "OPTIONS":
		echoServer.OPTIONS(fullpath, handler)
		return
	case "HEAD":
		echoServer.HEAD(fullpath, handler)
		return
	}

	panic("invalid method: " + e.GetMethod())
}

func resolveCachePolicy(endpointPolicy *HttpCachePolicy, response *Response) *HttpCachePolicy {
	if response.Headers.Get("Cache-Control") == "" {
		if response.CachePolicy != nil {
			return response.CachePolicy
		} else if endpointPolicy != nil {
			return endpointPolicy
		}
	}
	return nil
}
