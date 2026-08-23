package history

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/alexandre/senshi-training-planner/backend/internal/auth"
	"github.com/alexandre/senshi-training-planner/backend/internal/httpapi"
)

type Handler struct {
	service ServiceAPI
}

type completeRequest struct {
	ParticipantCount *int     `json:"participantCount"`
	ParticipantNames []string `json:"participantNames"`
}

func NewHandler(service ServiceAPI) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Collection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	h.list(w, r)
}

func (h *Handler) Resource(w http.ResponseWriter, r *http.Request) {
	segments := resourceSegments(r.URL.Path)
	if len(segments) != 1 {
		httpapi.WriteError(w, http.StatusNotFound, "history record not found")
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	h.get(w, r, segments[0])
}

func (h *Handler) CompleteSchedule(w http.ResponseWriter, r *http.Request, scheduleEntryID string) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httpapi.WriteError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	details, err := readCompleteRequest(w, r)
	if err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, "invalid request")
		return
	}

	detail, err := h.service.Complete(r.Context(), scheduleEntryID, user, details)
	writeCompletionResult(w, detail, err)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context(), r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	switch {
	case err == nil:
		httpapi.WriteJSON(w, http.StatusOK, items)
	case errors.Is(err, ErrInvalidRequest):
		httpapi.WriteError(w, http.StatusBadRequest, "invalid request")
	default:
		httpapi.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request, id string) {
	detail, err := h.service.Get(r.Context(), id)
	switch {
	case err == nil:
		httpapi.WriteJSON(w, http.StatusOK, detail)
	case errors.Is(err, ErrNotFound):
		httpapi.WriteError(w, http.StatusNotFound, "history record not found")
	default:
		httpapi.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeCompletionResult(w http.ResponseWriter, detail Detail, err error) {
	switch {
	case err == nil:
		httpapi.WriteJSON(w, http.StatusCreated, detail)
	case errors.Is(err, ErrScheduleNotFound):
		httpapi.WriteError(w, http.StatusNotFound, "schedule entry not found")
	case errors.Is(err, ErrAlreadyCompleted):
		httpapi.WriteError(w, http.StatusConflict, "training already completed")
	case errors.Is(err, ErrInvalidRequest):
		httpapi.WriteError(w, http.StatusBadRequest, "invalid request")
	case errors.Is(err, ErrSnapshotUnavailable):
		httpapi.WriteError(w, http.StatusInternalServerError, "internal server error")
	default:
		httpapi.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}

func readCompleteRequest(w http.ResponseWriter, r *http.Request) (CompletionDetails, error) {
	var request completeRequest
	if err := httpapi.ReadJSON(w, r, &request); err != nil {
		if errors.Is(err, io.EOF) {
			return CompletionDetails{}, nil
		}
		return CompletionDetails{}, err
	}

	return CompletionDetails{
		ParticipantCount: request.ParticipantCount,
		ParticipantNames: request.ParticipantNames,
	}, nil
}

func resourceSegments(path string) []string {
	resource := strings.TrimPrefix(path, "/history/")
	resource = strings.Trim(resource, "/")
	if resource == "" {
		return nil
	}

	return strings.Split(resource, "/")
}
