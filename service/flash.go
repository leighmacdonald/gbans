package service

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type Level string

const (
	lSuccess Level = "success"
	//lWarning Level = "warning"
	lError Level = "alert"
)

type Flash struct {
	Level   Level  `json:"level"`
	Heading string `json:"heading"`
	Message string `json:"message"`
}

func flash(c *gin.Context, level Level, heading string, msg string) {
	s := sessions.Default(c)
	s.AddFlash(Flash{
		Heading: heading,
		Level:   level,
		Message: msg,
	})
	if err := s.Save(); err != nil {
		log.Errorf("failed to save flash")
		return
	}
	log.Infof("Flashed: [%v] %s", level, msg)
}

func successFlash(c *gin.Context, heading string, msg string, path string) {
	flash(c, lSuccess, heading, msg)
	c.Redirect(http.StatusTemporaryRedirect, path)
}

func abortFlash(c *gin.Context, heading string, msg string, path string) {
	flash(c, lError, heading, msg)
	c.Redirect(http.StatusTemporaryRedirect, path)
}

func abortFlashErr(c *gin.Context, heading string, msg string, path string, err error) {
	abortFlash(c, msg, heading, path)
	log.Error(err)
}
