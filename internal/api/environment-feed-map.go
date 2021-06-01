package api

import (
	"gitlab.unanet.io/devops/eve/pkg/eve"
	"net/http"

	"gitlab.unanet.io/devops/eve/internal/service/crud"
	"gitlab.unanet.io/devops/go/pkg/json"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
)

type EnvironmentFeedMapController struct {
	manager *crud.Manager
}

func NewEnvironmentFeedMapController(manager *crud.Manager) *EnvironmentFeedMapController {
	return &EnvironmentFeedMapController{
		manager: manager,
	}
}

func (c EnvironmentFeedMapController) Setup(r chi.Router) {
	r.Get("/environment-feed-maps", c.environmentFeedMaps)
	r.Post("/environment-feed-maps", c.createEnvironmentFeedMaps)
	r.Put("/environment-feed-maps", c.updateEnvironmentFeedMap)
	r.Delete("/environment-feed-maps", c.deleteEnvironmentFeedMap)
}

func (c EnvironmentFeedMapController) environmentFeedMaps(w http.ResponseWriter, r *http.Request) {

	results, err := c.manager.EnvironmentFeedMaps(r.Context())

	if err != nil {
		render.Respond(w, r, err)
		return
	}

	render.Respond(w, r, results)
}

func (c EnvironmentFeedMapController) createEnvironmentFeedMaps(w http.ResponseWriter, r *http.Request) {

	var m eve.EnvironmentFeedMap
	if err := json.ParseBody(r, &m); err != nil {
		render.Respond(w, r, err)
		return
	}

	err := c.manager.CreateEnvironmentFeedMap(r.Context(), &m)
	if err != nil {
		render.Respond(w, r, err)
		return
	}

	render.Status(r, http.StatusCreated)
	render.Respond(w, r, m)
}

func (c EnvironmentFeedMapController) updateEnvironmentFeedMap(w http.ResponseWriter, r *http.Request) {

	var m eve.EnvironmentFeedMap
	if err := json.ParseBody(r, &m); err != nil {
		render.Respond(w, r, err)
		return
	}

	err := c.manager.UpdateEnvironmentFeedMap(r.Context(), &m)
	if err != nil {
		render.Respond(w, r, err)
		return
	}

	render.Status(r, http.StatusCreated)
	render.Respond(w, r, m)
}

func (c EnvironmentFeedMapController) deleteEnvironmentFeedMap(w http.ResponseWriter, r *http.Request) {

	var m eve.EnvironmentFeedMap
	if err := json.ParseBody(r, &m); err != nil {
		render.Respond(w, r, err)
		return
	}

	err := c.manager.DeleteEnvironmentFeedMap(r.Context(), &m)
	if err != nil {
		render.Respond(w, r, err)
		return
	}

	render.Status(r, http.StatusNoContent)
	render.Respond(w, r, m)
}
