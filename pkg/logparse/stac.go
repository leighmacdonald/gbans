package logparse

import (
	"bufio"
	"errors"
	"io"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	ErrReadLog       = errors.New("failed to read stac log contents")
	ErrParsePlayer   = errors.New("failed to parse player info")
	ErrParseSummary  = errors.New("failed to parse summary info")
	ErrParseFileName = errors.New("failed to parse file name")
	ErrParseTime     = errors.New("failed to parse time")
)

const splitMarker = "----------"

// Detection maps to the postgres enum `detection_type`.
type Detection string

const (
	Any                Detection = "any"
	Unknown            Detection = "unknown"
	SilentAim          Detection = "silent_aim"       // [StAC] SilentAim detection of 1.70° on Name.
	AimSnap            Detection = "aim_snap"         // [StAC] Aimsnap detection of 23.61° on Name.
	TooManyConnectiona Detection = "too_many_conn"    // [StAC] Too many connections from the same IP address %s from client %N
	Interp             Detection = "interp"           // [StAC] Player Name's interp was 500.0ms, indicating interp exploitation. Kicked from server.
	BHop               Detection = "bhop"             // [StAC] Player Nameq bhopped!
	CmdNumSpike        Detection = "cmdnum_spike"     // [StAC] Cmdnum SPIKE of 37 on mastro798.
	EyeAngles          Detection = "eye_angles"       // [StAC] Player cool man has invalid eye angles!
	InvalidUserCmd     Detection = "invalid_user_cmd" // [StAC] Player (2)DoesVac sent an invalid usercmd!
	OOBCVar            Detection = "oob_cvar"         // [StAC] [Detection] Player Name is cheating - OOB cvar/netvar value -1 on var cl_cmdrate!
	CheatCVar          Detection = "cheat_cvar"       // [StAC] [Detection] Player Name is cheating - detected known cheat var/concommand windows_speaker_config!
)

type StacEntry struct {
	AnticheatID int             `json:"anticheat_id"`
	SteamID     steamid.SteamID `json:"steam_id"`
	ServerID    int             `json:"server_id"`
	ServerName  string          `json:"server_name"`
	DemoID      *int            `json:"demo_id"`
	DemoName    string          `json:"demo_name"`
	DemoTick    int             `json:"demo_tick"`
	Name        string          `json:"name"`
	Detection   Detection       `json:"detection"`
	Summary     string          `json:"summary"`
	RawLog      string          `json:"raw_log"`
	CreatedOn   time.Time       `json:"created_on"`
}

// StacParser is responsible for parsing stac logs without going wild trying to parse everything
// as the structure is all over the place.
//
// Log name must match the reLogName regex, stac_mmddyy.log.
type StacParser struct {
	reSummary      *regexp.Regexp
	rePlayer       *regexp.Regexp
	rePlayerCached *regexp.Regexp
	reLogName      *regexp.Regexp
	reTime         *regexp.Regexp
	reDemo         *regexp.Regexp
	reStutter      *regexp.Regexp
}

func NewStacParser() StacParser {
	return StacParser{
		reSummary: regexp.MustCompile(`\[StAC]\s+(.+?)$`),
		rePlayer:  regexp.MustCompile(`Player: (?P<name>.+?)<(?P<pid>\d+)><(?P<sid>.+?)><(?P<team>(Unassigned|Red|Blue|Spectator|unknown))?>`),
		reLogName: regexp.MustCompile(`^stac_(\d{2})(\d{2})(\d{2}).log$`),
		reTime:    regexp.MustCompile(`^<(.+?)>`),
		// <05:28:30> Demo file: stv_demos/active/20240512-052232-pl_badwater.dem. Demo tick: 23861
		reDemo: regexp.MustCompile(`Demo file:.+?(\d+-\d+-.+?\.dem)\..+?Demo tick: (\d+)$`),
		// <08:40:49> Server framerate stuttered. Expected: ~66.6, got 5.
		reStutter:      regexp.MustCompile(`^<(\d{2}):(\d{2}):(\d{2})> Server framerate stuttered. Expected: ~(\d+.\d), got (.+?)\.`),
		rePlayerCached: regexp.MustCompile(`StAC cached SteamID: (.+?)$`),
	}
}

// Disabling OnPlayerRunCmd checks for 5.00 seconds.
func (p StacParser) Parse(logName string, reader io.Reader) ([]StacEntry, error) {
	// Get the date from the file name eg: stac_052224.log
	date, errDate := p.parseFileName(logName)
	if errDate != nil {
		return nil, errDate
	}

	var (
		entries []StacEntry
		current = StacEntry{CreatedOn: date}
	)
	scan := bufio.NewScanner(reader)
	for scan.Scan() {
		line := scan.Text()
		if strings.Contains(line, "hings will break") {
			continue
		}

		if line == splitMarker {
			if current.Summary != "" {
				if !current.SteamID.Valid() {
					slog.Debug("Anticheat entry had invalid steam_id", slog.String("raw", current.RawLog))
				} else {
					// If a summary is set, we know we have a previous entry to keep
					entries = append(entries, current)
				}
			}

			current = StacEntry{CreatedOn: date}

			continue
		}

		if strings.Contains(line, "<") {
			parsedTime, errTime := p.parseTime(line, current.CreatedOn)
			if errTime == nil {
				current.CreatedOn = parsedTime
			}
		}

		if strings.Contains(line, "[StAC] ") {
			match := p.reSummary.FindStringSubmatch(line)
			if len(match) != 2 || match[1] == "" {
				return nil, ErrParseSummary
			}
			current.Summary = match[1]
			current.Detection = p.parseDetection(line)
		}

		//
		if strings.Contains(line, " Player: ") {
			matches := p.rePlayer.FindStringSubmatch(line)
			current.SteamID = steamid.New(strings.Trim(matches[3], " "))
			current.Name = matches[1]
		}

		// Use the cached version of their steam id when the Player line contains STEAM_ID_PENDING, if available.
		// Player: HappyAlphaMale<976><STEAM_ID_PENDING><>.
		if strings.Contains(line, "StAC cached SteamID: ") && !current.SteamID.Valid() {
			matches := p.rePlayerCached.FindStringSubmatch(line)
			sid := steamid.New(strings.Trim(matches[1], " "))
			if sid.Valid() {
				current.SteamID = sid
			}

			if !current.SteamID.Valid() || current.Name == "" {
				return nil, ErrParsePlayer
			}
		}

		// <05:28:30> Demo file: stv_demos/active/20240512-052232-pl_badwater.dem. Demo tick: 23861
		if strings.Contains(line, "Demo file:") {
			matches := p.reDemo.FindStringSubmatch(line)
			if len(matches) == 3 {
				current.DemoName = matches[1]
				tick, errTick := strconv.Atoi(matches[2])
				if errTick != nil {
					slog.Warn("Failed to parse demo tick", slog.String("line", line))
				} else {
					current.DemoTick = tick
				}
			}
		}

		if line != "" && !strings.Contains(line, "hings will break") {
			current.RawLog += line + "\n"
		}
	}

	if current.Summary != "" {
		if !current.SteamID.Valid() {
			slog.Debug("Anticheat entry had invalid steam_id", slog.String("raw", current.RawLog))
		} else {
			entries = append(entries, current)
		}
	}

	return entries, nil
}

func (p StacParser) parseDetection(line string) Detection {
	switch {
	case strings.Contains(line, "[StAC] SilentAim"):
		return SilentAim
	case strings.Contains(line, "[StAC] Aimsnap"):
		return AimSnap
	case strings.Contains(line, "[StAC] Too many connections"):
		return TooManyConnectiona
	case strings.Contains(line, "interp exploitation"):
		return Interp
	case strings.Contains(line, "bhopped!"):
		return BHop
	case strings.Contains(line, "[StAC] Cmdnum SPIKE"):
		return CmdNumSpike
	case strings.Contains(line, "invalid eye angles"):
		return EyeAngles
	case strings.Contains(line, "sent an invalid usercmd!"):
		return InvalidUserCmd
	case strings.Contains(line, "OOB cvar/netvar value"):
		return OOBCVar
	case strings.Contains(line, "known cheat var/concommand"):
		return CheatCVar
	default:
		return Unknown
	}
}

// parseFileName transforms the log filename (eg: stac_052224.log) into a time.Time.
func (p StacParser) parseFileName(logName string) (time.Time, error) {
	value, errParse := time.Parse("stac_010206.log", logName)
	if errParse != nil {
		return time.Time{}, errors.Join(errParse, ErrParseFileName)
	}

	return value, nil
}

// parseTime transforms a log timestamp (eg: <01:13:00>) into a time.Time.
// This value is appended to the base date pulled from the filename to provide a unique identifier for the entry.
func (p StacParser) parseTime(line string, startTime time.Time) (time.Time, error) {
	match := p.reTime.FindStringSubmatch(line)
	if len(match) != 2 {
		return time.Time{}, ErrParseTime
	}

	hour, errHour := strconv.ParseInt(match[1], 10, 32)
	if errHour != nil {
		return time.Time{}, errors.Join(errHour, ErrParseTime)
	}

	minute, errMinute := strconv.ParseInt(match[2], 10, 32)
	if errMinute != nil {
		return time.Time{}, errors.Join(errMinute, ErrParseTime)
	}

	seconds, errSeconds := strconv.ParseInt(match[3], 10, 32)
	if errSeconds != nil {
		return time.Time{}, errors.Join(errSeconds, ErrParseTime)
	}

	total := (time.Hour * time.Duration(hour)) + (time.Minute * time.Duration(minute)) + (time.Second * time.Duration(seconds))

	return startTime.Add(total), nil
}
