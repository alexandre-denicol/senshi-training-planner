package workouts

import (
	"errors"
	"net/http"
	"strings"

	"github.com/alexandre/senshi-training-planner/backend/internal/httpapi"
)

type Handler struct {
	service ServiceAPI
}

func NewHandler(service ServiceAPI) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Collection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.list(w, r)
	case http.MethodPost:
		h.create(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) Resource(w http.ResponseWriter, r *http.Request) {
	segments := resourceSegments(r.URL.Path)
	if len(segments) == 0 {
		httpapi.WriteError(w, http.StatusNotFound, "workout not found")
		return
	}

	id := segments[0]
	if len(segments) == 1 {
		switch r.Method {
		case http.MethodGet:
			h.get(w, r, id)
		case http.MethodPut:
			h.update(w, r, id)
		case http.MethodDelete:
			h.delete(w, r, id)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	if len(segments) == 2 && segments[1] == "status" {
		if r.Method != http.MethodPatch {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		h.setStatus(w, r, id)
		return
	}

	httpapi.WriteError(w, http.StatusNotFound, "workout not found")
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	workouts, err := h.service.List(r.Context())
	if err != nil {
		httpapi.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, workouts)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request, id string) {
	workout, err := h.service.Get(r.Context(), id)
	writeWorkoutDetailResult(w, http.StatusOK, workout, err)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var input CreateInput
	if err := httpapi.ReadJSON(w, r, &input); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid request")
		return
	}

	workout, err := h.service.Create(r.Context(), input)
	writeWorkoutDetailResult(w, http.StatusCreated, workout, err)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request, id string) {
	var input UpdateInput
	if err := httpapi.ReadJSON(w, r, &input); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid request")
		return
	}

	workout, err := h.service.Update(r.Context(), id, input)
	writeWorkoutDetailResult(w, http.StatusOK, workout, err)
}

func (h *Handler) setStatus(w http.ResponseWriter, r *http.Request, id string) {
	var input StatusInput
	if err := httpapi.ReadJSON(w, r, &input); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid request")
		return
	}

	workout, err := h.service.SetStatus(r.Context(), id, input)
	writeWorkoutListResult(w, http.StatusOK, workout, err)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request, id string) {
	err := h.service.Delete(r.Context(), id)
	writeEmptyResult(w, err)
}

func writeWorkoutDetailResult(w http.ResponseWriter, successStatus int, workout WorkoutDetail, err error) {
	switch {
	case err == nil:
		httpapi.WriteJSON(w, successStatus, workout)
	case errors.Is(err, ErrInvalidRequest), errors.Is(err, ErrInvalidBlocks):
		httpapi.WriteError(w, http.StatusBadRequest, "invalid request")
	case errors.Is(err, ErrNameExists):
		httpapi.WriteError(w, http.StatusConflict, "workout name already exists")
	case errors.Is(err, ErrNotFound):
		httpapi.WriteError(w, http.StatusNotFound, "workout not found")
	default:
		httpapi.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeWorkoutListResult(w http.ResponseWriter, successStatus int, workout WorkoutListItem, err error) {
	switch {
	case err == nil:
		httpapi.WriteJSON(w, successStatus, workout)
	case errors.Is(err, ErrInvalidRequest):
		httpapi.WriteError(w, http.StatusBadRequest, "invalid request")
	case errors.Is(err, ErrNotFound):
		httpapi.WriteError(w, http.StatusNotFound, "workout not found")
	default:
		httpapi.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeEmptyResult(w http.ResponseWriter, err error) {
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, ErrInvalidRequest):
		httpapi.WriteError(w, http.StatusBadRequest, "invalid request")
	case errors.Is(err, ErrNotFound):
		httpapi.WriteError(w, http.StatusNotFound, "workout not found")
	default:
		httpapi.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}

func resourceSegments(path string) []string {
	resource := strings.TrimPrefix(path, "/workouts/")
	resource = strings.Trim(resource, "/")
	if resource == "" {
		return nil
	}

	return strings.Split(resource, "/")
}
