package model

import (
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/steamweb"
	log "github.com/sirupsen/logrus"
	"strconv"
	"time"
)

type MetaData map[string]any

// ServerEvent is a flat struct encapsulating a parsed log event
// Fields being present is event dependent, so do not assume everything will be
// available
type ServerEvent struct {
	LogID int64 `json:"log_id"`
	// Server is where the event happened
	Server    *Server            `json:"server"`
	EventType logparse.EventType `json:"event_type"`
	// Source is the player or thing initiating the event/action
	Source *Person `json:"source"`
	// Target is the optional target of an event created by a Source
	Target *Person `json:"target"`
	// PlayerClass is the last known class the player was as tracked by the playerStateCache OR the class that
	// a player switch to in the case of a spawned_as event
	PlayerClass logparse.PlayerClass `json:"player_class"`
	// Weapon is the weapon used to perform certain events
	Weapon logparse.Weapon `json:"weapon"`
	// Damage is how much (real) damage or in the case of medi-guns, healing
	Damage     int64 `json:"damage"`
	RealDamage int64 `json:"realdamage"`
	Healing    int64 `json:"healing"`
	// Item is the item a player picked up
	Item logparse.PickupItem `json:"item"`
	// AttackerPOS is the 3d position of the source of the event
	AttackerPOS logparse.Pos `json:"attacker_pos"`
	VictimPOS   logparse.Pos `json:"victim_pos"`
	AssisterPOS logparse.Pos `json:"assister_pos"`
	// Team is the last known team the player was as tracked by the playerStateCache OR the team that
	// a player switched to in a join_team event
	Team      logparse.Team     `json:"team"`
	CreatedOn time.Time         `json:"created_on"`
	Crit      logparse.CritType `json:"crit"`
	MetaData  MetaData          `json:"meta_data"`
}

func (m ServerEvent) GetValueAny(key string) any {
	v, found := m.MetaData[key]
	if !found {
		return nil
	}
	return v
}

func (m ServerEvent) GetValueString(key string) string {
	v, found := m.MetaData[key]
	if !found {
		return ""
	}
	return v.(string)
}

func (m ServerEvent) GetValueInt64(key string) int64 {
	v, found := m.MetaData[key]
	if !found {
		return 0
	}
	switch v.(type) {
	case string:
		value, errConv := strconv.ParseInt(v.(string), 10, 64)
		if errConv != nil {
			log.WithFields(log.Fields{"key": key}).Errorf("Failed to parse key value: %v", errConv)
		}
		return value
	case float64:
		return v.(int64)
	default:
		return v.(int64)
	}
}

func (m ServerEvent) GetValueInt(key string) int {
	return int(m.GetValueInt64(key))
}

func (m ServerEvent) GetValueBool(key string) bool {
	// TODO is there a stringy bool value?
	v, found := m.MetaData[key]
	if !found {
		return false
	}
	val, e := strconv.ParseBool(v.(string))
	if e != nil {
		log.Errorf("Failed to parse bool value: %v", e)
		return false
	}
	return val
}

func NewServerEvent() ServerEvent {
	return ServerEvent{
		Server:      &Server{},
		Source:      &Person{PlayerSummary: &steamweb.PlayerSummary{}},
		Target:      &Person{PlayerSummary: &steamweb.PlayerSummary{}},
		AssisterPOS: logparse.Pos{},
		AttackerPOS: logparse.Pos{},
		VictimPOS:   logparse.Pos{},
	}
}
