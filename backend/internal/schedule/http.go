package schedule

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
		httpapi.WriteError(w, http.StatusNotFound, "schedule entry not found")
		return
	}

	id := segments[0]
	if len(segments) == 1 {
		switch r.Method {
		case http.MethodPut:
			h.update(w, r, id)
		case http.MethodDelete:
			h.delete(w, r, id)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	httpapi.WriteError(w, http.StatusNotFound, "schedule entry not found")
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	entries, err := h.service.List(r.Context(), r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		writeEntryListResult(w, entries, err)
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, entries)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var input CreateInput
	if err := httpapi.ReadJSON(w, r, &input); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid request")
		return
	}

	entry, err := h.service.Create(r.Context(), input)
	writeEntryResult(w, http.StatusCreated, entry, err)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request, id string) {
	var input UpdateInput
	if err := httpapi.ReadJSON(w, r, &input); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid request")
		return
	}

	entry, err := h.service.Update(r.Context(), id, input)
	writeEntryResult(w, http.StatusOK, entry, err)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request, id string) {
	err := h.service.Delete(r.Context(), id)
	writeEmptyResult(w, err)
}

func writeEntryListResult(w http.ResponseWriter, entries []Entry, err error) {
	switch {
	case err == nil:
		httpapi.WriteJSON(w, http.StatusOK, entries)
	case errors.Is(err, ErrInvalidRequest):
		httpapi.WriteError(w, http.StatusBadRequest, "invalid request")
	default:
		httpapi.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeEntryResult(w http.ResponseWriter, successStatus int, entry Entry, err error) {
	switch {
	case err == nil:
		httpapi.WriteJSON(w, successStatus, entry)
	case errors.Is(err, ErrInvalidRequest), errors.Is(err, ErrInvalidWorkout):
		httpapi.WriteError(w, http.StatusBadRequest, "invalid request")
	case errors.Is(err, ErrDuplicate):
		httpapi.WriteError(w, http.StatusConflict, "workout already scheduled for date")
	case errors.Is(err, ErrNotFound):
		httpapi.WriteError(w, http.StatusNotFound, "schedule entry not found")
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
		httpapi.WriteError(w, http.StatusNotFound, "schedule entry not found")
	default:
		httpapi.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}

func resourceSegments(path string) []string {
	resource := strings.TrimPrefix(path, "/schedule/")
	resource = strings.Trim(resource, "/")
	if resource == "" {
		return nil
	}

	return strings.Split(resource, "/")
}
