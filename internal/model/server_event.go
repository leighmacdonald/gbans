package model

import (
	"github.com/leighmacdonald/gbans/pkg/logparse"
	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamweb"
	"github.com/pkg/errors"
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

func (serverEvent ServerEvent) GetValueInt64(key string) (int64, error) {
	value, found := serverEvent.MetaData[key]
	if !found {
		return 0, nil
	}
	switch v := value.(type) {
	case string:
		parsedValue, errConv := strconv.ParseInt(v, 10, 64)
		if errConv != nil {
			return 0, errors.New("Failed to parse key value")
		}
		return parsedValue, nil
	case float64:
		return value.(int64), nil
	default:
		return value.(int64), nil
	}
}

var ErrInvalidKey = errors.New("invalid key")

func (serverEvent ServerEvent) GetValueInt(key string) (int, error) {
	value, errGet := serverEvent.GetValueInt64(key)
	if errGet != nil {
		return 0, errGet
	}
	if value > 0 && value <= math.MaxInt32 {
		return int(value), nil
	}
	return util.DefaultIntAllocate, nil
}

func (serverEvent ServerEvent) GetValueBool(key string) (bool, error) {
	// TODO is there a stringy bool value?
	value, found := serverEvent.MetaData[key]
	if !found {
		return false, ErrInvalidKey
	}
	val, errParse := strconv.ParseBool(value.(string))
	if errParse != nil {
		return false, errParse
	}
	return val, nil
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
