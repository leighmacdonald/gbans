#pragma semicolon 1
#pragma tabsize 4
#pragma newdecls required

#define PLUGIN_NAME "gbans"
#define PLUGIN_VERSION "0.7.47"

#define MAX_SCORES 256

bool gLateLoaded;

bool gPlayerStatus[MAXPLAYERS + 1];

// Core gbans options
ConVar gbCoreHost;
ConVar gbCorePort;
ConVar gbCoreServerKey;

// In Game Tweaks
ConVar gbDisableAutoteam;
ConVar gbHideConnections;

// STV options
ConVar gbStvEnable;
ConVar gbAutoRecord;
ConVar gbStvMinplayers;
ConVar gbStvIgnorebots;
ConVar gbStvTimestart;
ConVar gbStvTimestop;
ConVar gbStvFinishmap;
ConVar gbStvPath;
ConVar gbStvPathComplete;

// Reports command
int gReportSourceId = -1;
int gReportTargetId = -1;
bool gReportWaitingForReason = false;
GB_BanReason gReportTargetReason;
int gReportStartedAtTime = -1;

// Stv
bool gStvMapChanged = false;
bool gIsRecording = false;
bool gIsManual = false;

// jwt returned and used once authenticated
char gToken[1024] = "";
bool gAuthWaiting = false;
//int gLastAuthAttempt = 0;
