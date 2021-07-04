package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/render"

	"gitlab.unanet.io/devops/go/pkg/errors"
)

type PingController struct {
}

func NewPingController() *PingController {
	return &PingController{}
}

func (c PingController) Setup(r *Routers) {
	r.Anonymous.Get("/internal-error", c.internalError)
	r.Anonymous.Get("/rest-error", c.restError)
	r.Anonymous.Get("/ping", c.ping)
}

func (c PingController) restError(w http.ResponseWriter, r *http.Request) {
	render.Respond(w, r, errors.RestError{
		Code:          400,
		Message:       "Bad Request",
		OriginalError: nil,
	})
}

func (c PingController) internalError(w http.ResponseWriter, r *http.Request) {
	render.Respond(w, r, fmt.Errorf("some error"))
}

func (c PingController) ping(w http.ResponseWriter, r *http.Request) {
	render.Respond(w, r, render.M{
		"message": "pong",
		"version": Version,
	})
}
