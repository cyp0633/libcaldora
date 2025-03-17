package server

import (
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"

	"github.com/cyp0633/libcaldora/davserver/interfaces"
)

func (s *Server) sendError(w http.ResponseWriter, err error) {
	var httpErr *interfaces.HTTPError
	if !errors.As(err, &httpErr) {
		httpErr = &interfaces.HTTPError{Status: http.StatusInternalServerError, Message: err.Error()}
	}

	s.logger.Error("error response",
		"status", httpErr.Status,
		"message", httpErr.Message,
		"error", httpErr.Err)

	errResp := &interfaces.ErrorResponse{
		Message: httpErr.Message,
	}

	body, err := xml.Marshal(errResp)
	if err != nil {
		s.logger.Error("failed to marshal error response", "error", err)
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	w.WriteHeader(httpErr.Status)
	_, _ = w.Write(body)
}
