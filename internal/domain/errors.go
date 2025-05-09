package domain

import (
	"errors"
)

var (
	ErrMessageContext           = errors.New("could not fetch message context")
	ErrFailedWeapon             = errors.New("failed to save weapon")
	ErrSaveChanges              = errors.New("cannot save changes")
	ErrDiscordAlreadyLinked     = errors.New("discord account is already linked")
	ErrSaveBan                  = errors.New("failed to save ban")
	ErrReportStateUpdate        = errors.New("failed to update report state")
	ErrFetchPerson              = errors.New("failed to fetch/create person")
	ErrFetchSource              = errors.New("failed to fetch source player")
	ErrFetchTarget              = errors.New("failed to fetch target player")
	ErrGroupValidate            = errors.New("failed to validate group")
	ErrParseASN                 = errors.New("failed to parse asn")
	ErrFetchASNBan              = errors.New("failed to fetch asn ban")
	ErrDropASNBan               = errors.New("failed to drop existing asn ban")
	ErrInvalidPattern           = errors.New("invalid pattern")
	ErrNetworkInvalidIP         = errors.New("invalid ip")
	ErrNetworkLocationUnknown   = errors.New("unknown location record")
	ErrNetworkASNUnknown        = errors.New("unknown asn record")
	ErrNetworkProxyUnknown      = errors.New("no proxy record")
	ErrSteamAPIArgLimit         = errors.New("steam api support a max of 100 steam ids")
	ErrFetchSteamBans           = errors.New("failed to fetch ban status from steam api")
	ErrSteamAPISummaries        = errors.New("failed to fetch player summaries")
	ErrSteamAPI                 = errors.New("steam api requests have errors")
	ErrUpdatePerson             = errors.New("failed to save updated person profile")
	ErrCommandFailed            = errors.New("command failed")
	ErrDiscordCreate            = errors.New("failed to connect to discord")
	ErrDiscordOpen              = errors.New("failed to open discord connection")
	ErrDuplicateCommand         = errors.New("duplicate command registration")
	ErrDiscordMessageSen        = errors.New("failed to send discord message")
	ErrDiscordOverwriteCommands = errors.New("failed to bulk overwrite discord commands")
	ErrInsufficientPlayers      = errors.New("insufficient Match players")
	ErrIncompleteMatch          = errors.New("insufficient match data")
	ErrSaveMatch                = errors.New("could not save match results")
	ErrLoadMatch                = errors.New("could not load match results")
	ErrLoadServer               = errors.New("failed to load match server")
	ErrMissingParam             = errors.New("failed to request at least one required parameter")
	ErrBanDoesNotExist          = errors.New("ban does not exist")
	ErrSteamUnset               = errors.New("must connect discord, see connections in your settings page to connect it")
	ErrFetchClassStats          = errors.New("failed to fetch class stats")
	ErrFetchWeaponStats         = errors.New("failed to fetch weapon stats")
	ErrFetchKillstreakStats     = errors.New("failed to fetch killstreak stats")
	ErrFetchMedicStats          = errors.New("failed to fetch medic stats")
	ErrGetServer                = errors.New("failed to get server")
	ErrReasonInvalid            = errors.New("invalid reason")
	ErrDuplicateBan             = errors.New("duplicate ban")
	ErrCIDRMissing              = errors.New("cidr invalid or missing")
	ErrRowResults               = errors.New("resulting rows contain error")
	ErrTxStart                  = errors.New("could not start transaction")
	ErrTxCommit                 = errors.New("failed to commit tx changes")
	ErrTxRollback               = errors.New("could not rollback transaction")
	ErrPoolFailed               = errors.New("could not create store pool")
	ErrUUIDGen                  = errors.New("could not generate uuid")
	ErrCreateQuery              = errors.New("failed to generate query")
	ErrCountQuery               = errors.New("failed to get count result")
	ErrTooShort                 = errors.New("value too short")
	ErrInvalidParameter         = errors.New("invalid parameter format")
	ErrPermissionDenied         = errors.New("permission denied")
	ErrBadRequest               = errors.New("invalid request")
	ErrInternal                 = errors.New("internal server error")
	ErrParamKeyMissing          = errors.New("param key not found")
	ErrParamParse               = errors.New("failed to parse param value")
	ErrParamInvalid             = errors.New("param value invalid")
	ErrScanResult               = errors.New("failed to scan result")
	ErrUnknownServerID          = errors.New("unknown server id")
	ErrSelfReport               = errors.New("cannot self report")
	ErrUUIDCreate               = errors.New("failed to generate new uuid")
	ErrUUIDInvalid              = errors.New("invalid uuid")
	ErrReportExists             = errors.New("duplicate user report")
	ErrEmptyToken               = errors.New("invalid Access token decoded")
	ErrContestMaxEntries        = errors.New("entries count exceed max_submission limits")
	ErrThreadLocked             = errors.New("thread is locked")
	ErrCreateToken              = errors.New("failed to create new Access token")
	ErrClientIP                 = errors.New("failed to parse IP")
	ErrSaveToken                = errors.New("failed to save new createRefresh token")
	ErrSignToken                = errors.New("failed create signed string")
	ErrAuthHeader               = errors.New("failed to bind auth header")
	ErrMalformedAuthHeader      = errors.New("invalid auth header format")
	ErrCookieKeyMissing         = errors.New("cookie key missing, cannot generate token")
	ErrInvalidContestID         = errors.New("invalid contest id")
	ErrInvalidDescription       = errors.New("invalid description, cannot be empty")
	ErrTitleEmpty               = errors.New("title cannot be empty")
	ErrDescriptionEmpty         = errors.New("description cannot be empty")
	ErrEndDateBefore            = errors.New("end date comes before start date")
	ErrInvalidThread            = errors.New("invalid thread id")
	ErrPersonSource             = errors.New("failed to load source person")
	ErrPersonTarget             = errors.New("failed to load target person")
	ErrGetBan                   = errors.New("failed to load existing ban")
	ErrScanASN                  = errors.New("failed to scan asn result")
	ErrCloseBatch               = errors.New("failed to close batch request")
	ErrDecodeDuration           = errors.New("failed to decode duration")
	ErrReadConfig               = errors.New("failed to read config file")
	ErrFormatConfig             = errors.New("config file format invalid")
	ErrSteamAPIKey              = errors.New("failed to set steam api key")
	ErrUnbanFailed              = errors.New("failed to perform unban")
	ErrStateUnchanged           = errors.New("state must be different than previous")
	ErrInvalidRegex             = errors.New("invalid regex format")
	ErrInvalidWeight            = errors.New("invalid weight value")
	ErrMatchQuery               = errors.New("failed to load match")
	ErrQueryPlayers             = errors.New("failed to query match players")
	ErrQueryMatch               = errors.New("failed to query match")
	ErrChatQuery                = errors.New("failed to query chat history")
	ErrGetPlayerClasses         = errors.New("failed to fetch player class stats")
	ErrGetMedicStats            = errors.New("failed to fetch medic class stats")
	ErrSaveMedicStats           = errors.New("failed to save medic stats")
	ErrSavePlayerStats          = errors.New("failed to save player stats")
	ErrSaveWeaponStats          = errors.New("failed to save weapon stats")
	ErrSaveClassStats           = errors.New("failed to save class stats")
	ErrSaveKillstreakStats      = errors.New("failed to save killstreak stats")
	ErrGetWeaponStats           = errors.New("failed to fetch match weapon stats")
	ErrGetPlayerKillstreaks     = errors.New("failed to fetch player killstreak stats")
	ErrGetPerson                = errors.New("failed to fetch person result")
	ErrInvalidIP                = errors.New("invalid ip, could not parse")
	ErrAuthentication           = errors.New("auth invalid")
	ErrExpired                  = errors.New("expired")
	ErrInvalidSID               = errors.New("invalid steamid")
	ErrInvalidConfig            = errors.New("invalid config value")
	ErrSourceID                 = errors.New("invalid source steam id")
	ErrTargetID                 = errors.New("invalid target steam id")
	ErrPlayerNotFound           = errors.New("could not find player")
	ErrUnknownID                = errors.New("could not find matching server/player/steamid")
	ErrInvalidAuthorSID         = errors.New("invalid author steam id")
	ErrInvalidTargetSID         = errors.New("invalid target steam id")
	ErrNotFound                 = errors.New("entity not found")
	ErrNoResult                 = errors.New("no results found")
	ErrDuplicate                = errors.New("entity already exists")
	ErrUnknownServer            = errors.New("unknown server")
	ErrVoteDeleted              = errors.New("vote deleted")
	ErrRequestCreate            = errors.New("failed to create new request")
	ErrRequestPerform           = errors.New("could not perform http request")
	ErrRequestInvalidCode       = errors.New("invalid response code returned from request")
	ErrRequestDecode            = errors.New("failed to decode http response")
	ErrResponseBody             = errors.New("failed to read response body")
	ErrQueryPatreon             = errors.New("failed to query patreon")
	ErrMimeTypeNotAllowed       = errors.New("mimetype is not allowed")
	ErrMimeTypeReadFailed       = errors.New("failed to read mime type")
	ErrCopyFileContent          = errors.New("could not copy read contents")
	ErrHashFileContent          = errors.New("could not hash reader bytes")
	ErrCreateAssetPath          = errors.New("failed to create asset path")
	ErrDeleteAssetFile          = errors.New("failed to remove asset from local store")
	ErrCreateAddFile            = errors.New("failed to create asset on filesystem")
	ErrAssetName                = errors.New("invalid asset name")
	ErrBucketType               = errors.New("invalid bucket type")
	ErrAssetTooLarge            = errors.New("asset exceeds max allowed size")
	ErrWarnActionApply          = errors.New("failed to apply warning action")
	ErrStaticPathError          = errors.New("could not load static path")
	ErrDataUpdate               = errors.New("data update failed")
	ErrValidateURL              = errors.New("could not validate url")
	ErrOpenFile                 = errors.New("could not open output file")
	ErrFrontendRoutes           = errors.New("failed to initialize frontend asset routes")
	ErrPathInvalid              = errors.New("invalid path specified")
	ErrDemoLoad                 = errors.New("could not load demo file")
	ErrValueOutOfRange          = errors.New("value out of range")
)
