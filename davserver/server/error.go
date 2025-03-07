package server

import (
	"encoding/xml"
	"errors"
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

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(httpErr.Status)

	errResp := &interfaces.ErrorResponse{
		Namespace: "DAV:",
		Message:   httpErr.Message,
	}

	enc := xml.NewEncoder(w)
	if err := enc.Encode(errResp); err != nil {
		s.logger.Error("failed to encode error response",
			"error", err)
	}
}
