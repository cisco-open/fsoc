package api

import (
	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"io"
)

type Handler struct {
	origHandler log.Handler
	level       log.Level
}

func New(w io.Writer, level log.Level) *Handler {
	originalHandler := cli.New(w)
	return &Handler{
		origHandler: originalHandler,
		level:       level,
	}
}

func (h *Handler) HandleLog(e *log.Entry) error {
	if e.Level >= h.level {
		return h.origHandler.HandleLog(e)
	}
	return nil
}
