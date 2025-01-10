package demostats

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	ErrDemoRead           = errors.New("could not read demo file")
	ErrDemoRequestInit    = errors.New("could not create request")
	ErrDemoRequestPerform = errors.New("error performing request")
	ErrDemoParseResponse  = errors.New("error parsing response")
)

type Stats struct {
	DemoType        string            `json:"demo_type"`
	Version         int               `json:"version"`
	Protocol        int               `json:"protocol"`
	Server          string            `json:"server"`
	Nick            string            `json:"nick"`
	Map             string            `json:"map"`
	Game            string            `json:"game"`
	Duration        float64           `json:"duration"`
	Ticks           int               `json:"ticks"`
	Frames          int               `json:"frames"`
	Signon          int               `json:"signon"`
	PlayerSummaries map[string]Player `json:"player_summaries"`
}

type Team string

const (
	TeamOther     = "other"
	TeamSpectator = "spectator"
	TeamRed       = "red"
	TeamBlue      = "blu"
)

type Player struct {
	Name               string       `json:"name"`
	SteamID            steamid.SID3 `json:"steamid"` //nolint:tagliatelle
	Team               Team         `json:"team"`
	TimeStart          int          `json:"time_start"`
	TimeEnd            int          `json:"time_end"`
	Points             int          `json:"points"`
	ConnectionCount    int          `json:"connection_count"`
	BonusPoints        int          `json:"bonus_points"`
	Kills              int          `json:"kills"`
	ScoreboardKills    int          `json:"scoreboard_kills"`
	PostroundKills     int          `json:"postround_kills"`
	Assists            int          `json:"assists"`
	ScoreboardAssists  int          `json:"scoreboard_assists"`
	PostroundAssists   int          `json:"postround_assists"`
	Suicides           int          `json:"suicides"`
	Deaths             int          `json:"deaths"`
	ScoreboardDeaths   int          `json:"scoreboard_deaths"`
	PostroundDeaths    int          `json:"postround_deaths"`
	Defenses           int          `json:"defenses"`
	Dominations        int          `json:"dominations"`
	Dominated          int          `json:"dominated"`
	Revenges           int          `json:"revenges"`
	Damage             int          `json:"damage"`
	DamageTaken        int          `json:"damage_taken"`
	HealingTaken       int          `json:"healing_taken"`
	HealthPacks        int          `json:"health_packs"`
	HealingPacks       int          `json:"healing_packs"`
	Captures           int          `json:"captures"`
	CapturesBlocked    int          `json:"captures_blocked"`
	Extinguishes       int          `json:"extinguishes"`
	BuildingBuilt      int          `json:"building_built"`
	BuildingsDestroyed int          `json:"buildings_destroyed"`
	Airshots           int          `json:"airshots"`
	Ubercharges        int          `json:"ubercharges"`
	Headshots          int          `json:"headshots"`
	Shots              int          `json:"shots"`
	Hits               int          `json:"hits"`
	Teleports          int          `json:"teleports"`
	Backstabs          int          `json:"backstabs"`
	Support            int          `json:"support"`
	DamageDealt        int          `json:"damage_dealt"`
	Healing            Healing      `json:"healing"`
	Classes            struct{}     `json:"classes"`
	Killstreaks        []Killstreak `json:"killstreaks"`
	Weapons            Weapons      `json:"weapons"`
}

var ErrPlayerNotFound = errors.New("player not found")

func (s Stats) Player(steamID steamid.SteamID) (Player, string, error) {
	for uid, player := range s.PlayerSummaries {
		if player.SteamID == steamID.Steam3() {
			return player, uid, nil
		}
	}

	return Player{}, "", ErrPlayerNotFound
}

type Weapons struct {
	WeaponID int `json:"weapon_id"`
}

type Killstreak struct {
	Count int `json:"count"`
}

type Healing struct {
	Healing             int `json:"healing"`
	ChargesUber         int `json:"charges_uber"`
	ChargesKritz        int `json:"charges_kritz"`
	ChargesVacc         int `json:"charges_vacc"`
	ChargesQuickfix     int `json:"charges_quickfix"`
	Drops               int `json:"drops"`
	NearFullChargeDeath int `json:"near_full_charge_death"`
	AvgUberLength       int `json:"avg_uber_length"`
	MajorAdvLost        int `json:"major_adv_lost"`
	BiggestAdvLost      int `json:"biggest_adv_lost"`
}

func Submit(ctx context.Context, url string, demoPath string) (Stats, error) {
	fileHandle, errDF := os.Open(demoPath)
	if errDF != nil {
		return Stats{}, errors.Join(errDF, ErrDemoRead)
	}

	content, errContent := io.ReadAll(fileHandle)
	if errContent != nil {
		return Stats{}, errors.Join(errDF, ErrDemoRead)
	}

	info, errInfo := fileHandle.Stat()
	if errInfo != nil {
		return Stats{}, errors.Join(errInfo, ErrDemoRead)
	}

	log.Closer(fileHandle)

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	part, errCreate := writer.CreateFormFile("file", info.Name())
	if errCreate != nil {
		return Stats{}, errors.Join(errCreate, ErrDemoRequestInit)
	}

	if _, err := part.Write(content); err != nil {
		return Stats{}, errors.Join(errCreate, ErrDemoRequestInit)
	}

	if errClose := writer.Close(); errClose != nil {
		return Stats{}, errors.Join(errClose, ErrDemoRequestInit)
	}

	req, errReq := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if errReq != nil {
		return Stats{}, errors.Join(errReq, ErrDemoRequestPerform)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, errSend := client.Do(req) //nolint:bodyclose
	if errSend != nil {
		return Stats{}, errors.Join(errSend, ErrDemoRequestPerform)
	}

	defer log.Closer(resp.Body)

	return ParseReader(resp.Body)
}

func ParseReader(jsonReader io.Reader) (Stats, error) {
	// TODO remove this extra copy once this feature doesnt have much need for debugging/inspection.
	rawBody, errRead := io.ReadAll(jsonReader)
	if errRead != nil {
		return Stats{}, errors.Join(errRead, ErrDemoParseResponse)
	}

	var stats Stats
	if errDecode := json.NewDecoder(bytes.NewReader(rawBody)).Decode(&stats); errDecode != nil {
		return Stats{}, errors.Join(errDecode, ErrDemoParseResponse)
	}

	return stats, nil
}
