package depotdownloader

import (
	"github.com/gorilla/websocket"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net/url"
	"time"
)

type Depot struct {
	AppID   steamid.AppID
	Version uint64
	Date    time.Time
}

type swpPayloadType string

const (
	usersOnline swpPayloadType = "UsersOnline"
	logOn       swpPayloadType = "LogOn"
	logOff      swpPayloadType = "LogOff"
	changelist  swpPayloadType = "Changelist"
)

type swpPayload struct {
	ChangeNumber int                      `json:"ChangeNumber,omitempty"`
	Apps         map[steamid.AppID]string `json:"Apps,omitempty"`
	Packages     struct{}                 `json:"Packages,omitempty"`
	Type         swpPayloadType           `json:"Type"`
	UsersOnline  int                      `json:"Users,omitempty"`
}

type onAppUpdate = func(depot Depot) error

func VersionChangeListener(u *url.URL, appid steamid.AppID, onAppUpdate onAppUpdate) error {
	c, _, errConn := websocket.DefaultDialer.Dial(u.String(), nil)
	if errConn != nil {
		return errors.Wrapf(errConn, "Failed to connect")
	}
	defer c.Close()
	for {
		if errDL := c.SetReadDeadline(time.Now().Add(time.Second * 60)); errDL != nil {
			return errors.Wrapf(errDL, "Failed to set read deadline")
		}
		var p swpPayload
		errRead := c.ReadJSON(&p)
		if errRead != nil {
			return errors.Wrapf(errRead, "Failed to read payload")
		}
		switch p.Type {
		case usersOnline:
			log.WithFields(log.Fields{"count": p.UsersOnline}).Debugf("Users online")
		case logOn:
			log.Debugf("Client connected: %v", p)
		case logOff:
			log.Debugf("Client disconnected: %v", p)
		case changelist:
			for aid := range p.Apps {
				log.WithFields(log.Fields{"app_id": aid, "version": p.ChangeNumber}).Infof("App update")
				if aid == appid {
					go func() {
						if errUpd := onAppUpdate(Depot{
							AppID:   aid,
							Version: uint64(p.ChangeNumber),
							Date:    time.Now(),
						}); errUpd != nil {
							log.WithFields(log.Fields{"app_id": aid, "version": p.ChangeNumber}).
								Error("Failed to execute update handler")
						}
					}()
				}
			}
		}
	}
}
