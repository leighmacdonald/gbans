#if defined _gbans_globals_included
 #endinput
#endif
#define _gbans_globals_included

#define PLUGIN_AUTHOR "Leigh MacDonald"
#define PLUGIN_VERSION "0.00"
#define PLUGIN_NAME "gbans"

#define MAX_SCORES 256

// clang-format off
enum struct PlayerInfo {
    bool authed;
    char ip[16];
    int banType;
    int permissionLevel;
    char message[256];
}
// clang-format on


// Globals must all start with g
PlayerInfo gPlayers[MAXPLAYERS + 1];

// Core gbans options
ConVar gPort  = null;
ConVar gHost  = null;
ConVar gServerName = null;
ConVar gServerKey = null;

// STV options
ConVar gTvEnabled = null;
ConVar gAutoRecord = null;
ConVar gMinPlayersStart = null;
ConVar gIgnoreBots = null;
ConVar gTimeStart = null;
ConVar gTimeStop = null;
ConVar gFinishMap = null;
ConVar gDemoPathActive = null;
ConVar gDemoPathComplete = null;


// Stopwatch options
ConVar gStopwatchEnabled = null;
ConVar gStopwatchNameRed = null;
ConVar gStopwatchNameBlu = null;

char gAccessToken[512];

// Store temp clientId for networked callbacks 
int gReplyToClientId = 0;

// Reports command
bool gReportInProgress = false;
char gReportSid64[30];
char gReportReasonCustom[1024];
GB_BanReason gReportTargetReason;

// Stv

bool gIsRecording = false;
bool gIsManual = false;
JSON_Object gScores = null;
