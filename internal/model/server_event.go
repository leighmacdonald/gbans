package model

import (
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamweb"
	log "github.com/sirupsen/logrus"
	"math"
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
	Server    Server             `json:"server"`
	EventType logparse.EventType `json:"event_type"`
	// Source is the player or thing initiating the event/action
	Source Person `json:"source"`
	// Target is the optional target of an event created by a Source
	Target Person `json:"target"`
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

func (serverEvent ServerEvent) GetValueAny(key string) any {
	value, found := serverEvent.MetaData[key]
	if !found {
		return nil
	}
	return value
}

func (serverEvent ServerEvent) GetValueString(key string) string {
	value, found := serverEvent.MetaData[key]
	if !found {
		return ""
	}
	return value.(string)
}

func (serverEvent ServerEvent) GetValueInt64(key string) int64 {
	value, found := serverEvent.MetaData[key]
	if !found {
		return 0
	}
	switch v := value.(type) {
	case string:
		parsedValue, errConv := strconv.ParseInt(v, 10, 64)
		if errConv != nil {
			log.WithFields(log.Fields{"key": key}).Errorf("Failed to parse key value: %value", errConv)
		}
		return parsedValue
	case float64:
		return value.(int64)
	default:
		return value.(int64)
	}
}

func (serverEvent ServerEvent) GetValueInt(key string) int {
	value := serverEvent.GetValueInt64(key)
	if value > 0 && value <= math.MaxInt32 {
		return int(value)
	}
	return util.DefaultIntAllocate
}

func (serverEvent ServerEvent) GetValueBool(key string) bool {
	// TODO is there a stringy bool value?
	value, found := serverEvent.MetaData[key]
	if !found {
		return false
	}
	val, errParse := strconv.ParseBool(value.(string))
	if errParse != nil {
		log.Errorf("Failed to parse bool value: %value", errParse)
		return false
	}
	return val
}

func NewServerEvent() ServerEvent {
	return ServerEvent{
		Server:      Server{},
		Source:      Person{PlayerSummary: &steamweb.PlayerSummary{}},
		Target:      Person{PlayerSummary: &steamweb.PlayerSummary{}},
		AssisterPOS: logparse.Pos{},
		AttackerPOS: logparse.Pos{},
		VictimPOS:   logparse.Pos{},
	}
}
