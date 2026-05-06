#pragma semicolon 1
#pragma tabsize 4
#pragma newdecls required

#include <admin>
#include <adminmenu>
#include <basecomm>
#include <gbans>
#include <sdktools>
#include <ripext>
#include <sourcemod>
#include <autoexecconfig>
#include <sourcetvmanager>
#include <tf2_stocks>

#include "gbans/globals.sp"
#include "gbans/admins.sp"
#include "gbans/auth.sp"
#include "gbans/balance.sp"
#include "gbans/ban.sp"
#include "gbans/commands.sp"
#include "gbans/common.sp"
#include "gbans/connect.sp"
#include "gbans/report.sp"
#include "gbans/stv.sp"

public Plugin myinfo =
{
	name = PLUGIN_NAME,
	author = "Leigh MacDonald",
	description = "gbans game client",
	version = PLUGIN_VERSION,
	url = "https://github.com/leighmacdonald/gbans",
};


public void OnPluginStart()
{
	LoadTranslations("common.phrases.txt");

	RegConsoleCmd("gb_version", onCmdVersion, "Get gbans version");
	RegConsoleCmd("gb_help", onCmdHelp, "Get a list of gbans commands");
	RegConsoleCmd("gb_mod", onCmdMod, "Ping a moderator");
	RegConsoleCmd("mod", onCmdMod, "Ping a moderator");
	RegConsoleCmd("seed", onCmdSeed, "Send a seed request to discord");

	RegConsoleCmd("report", onCmdReport, "Report a player");
	RegConsoleCmd("autoteam", onCmdAutoTeamAction);

	RegAdminCmd("gb_ban", onAdminCmdBan, ADMFLAG_BAN);
	RegAdminCmd("gb_reload", onAdminCmdReload, ADMFLAG_ROOT);

	RegAdminCmd("gb_stv_record", Command_Record, ADMFLAG_KICK, "Starts a SourceTV demo");
	RegAdminCmd("gb_stv_stoprecord", Command_StopRecord, ADMFLAG_KICK, "Stops the current SourceTV demo");

	HookEvent("player_disconnect", Event_PlayerDisconnect, EventHookMode_Pre);
	HookEvent("player_connect_client", Event_PlayerConnect, EventHookMode_Pre);

	AutoExecConfig_SetFile("gbans");

	// Core settings
	gbCoreHost = AutoExecConfig_CreateConVar("gb_core_host", "localhost", "Remote gbans host", FCVAR_NONE);
	gbCorePort = AutoExecConfig_CreateConVar("gb_core_port", "6006", "Remote gbans port", FCVAR_NONE, true, 1.0, true, 65535.0);
	gbCoreServerKey = AutoExecConfig_CreateConVar("gb_core_server_key", "", "GBans server key used to authenticate with the service", FCVAR_NONE);

	// In Game Tweaks
	gbDisableAutoteam = AutoExecConfig_CreateConVar("gb_hide_connections", "1", "Dont allow the use of autoteam command", FCVAR_NONE, true, 0.0, true, 1.0);
	gbHideConnections = AutoExecConfig_CreateConVar("gb_disable_autoteam", "1", "Dont show the disconnect message to users", FCVAR_NONE, true, 0.0, true, 1.0);

	// STV settings
	gbStvEnable = AutoExecConfig_CreateConVar("gb_stv_enable", "1", "Enable SourceTV", FCVAR_NONE, true, 0.0, true, 1.0);
	gbAutoRecord = AutoExecConfig_CreateConVar("gb_auto_record", "1", "Enable automatic recording", FCVAR_NONE, true, 0.0, true, 1.0);
	gbStvMinplayers = AutoExecConfig_CreateConVar("gb_stv_minplayers", "1", "Minimum players on server to start recording", _, true, 0.0);
	gbStvIgnorebots = AutoExecConfig_CreateConVar("gb_stv_ignorebots", "1", "Ignore bots in the player count", FCVAR_NONE, true, 0.0, true, 1.0);
	gbStvTimestart = AutoExecConfig_CreateConVar("gb_stv_timestart", "-1", "Hour in the day to start recording (0-23, -1 disables)", FCVAR_NONE);
	gbStvTimestop = AutoExecConfig_CreateConVar("gb_stv_timestop", "-1", "Hour in the day to stop recording (0-23, -1 disables)", FCVAR_NONE);
	gbStvFinishmap = AutoExecConfig_CreateConVar("gb_stv_finishmap", "1", "If 1, continue recording until the map ends", FCVAR_NONE, true, 0.0, true, 1.0);
	gbStvPath = AutoExecConfig_CreateConVar("gb_stv_path", "stv_demos/active", "Path to store currently recording demos", FCVAR_NONE);
	gbStvPathComplete = AutoExecConfig_CreateConVar("gb_stv_path_complete", "stv_demos/complete", "Path to store complete demos", FCVAR_NONE);

	AutoExecConfig_ExecuteFile();
	AutoExecConfig_CleanFile();

	//BuildPath(Path_SM, logFile, sizeof(logFile), "logs/gbans.log");

	if(gLateLoaded)
	{
		AccountForLateLoading();
	}

	authenticateServer();
}


stock void AccountForLateLoading()
{
	char auth[30];

	for(int i = 1; i <= MaxClients; i++)
	{
		if(IsClientConnected(i) && !IsFakeClient(i))
		{
			gPlayerStatus[i] = false;
		}
		if(IsClientInGame(i) && !IsFakeClient(i) && IsClientAuthorized(i) && GetClientAuthId(i, AuthId_Steam2, auth, sizeof auth))
		{
			checkPlayer(i);
		}
	}
}


public void OnConfigsExecuted()
{
	gbStvMinplayers.AddChangeHook(OnConVarChanged);
	gbStvIgnorebots.AddChangeHook(OnConVarChanged);
	gbStvTimestart.AddChangeHook(OnConVarChanged);
	gbStvTimestop.AddChangeHook(OnConVarChanged);
	gbStvPath.AddChangeHook(OnConVarChanged);

	char sPath[PLATFORM_MAX_PATH];

	gbStvPath.GetString(sPath, sizeof sPath);
	if(!DirExists(sPath))
	{
		initDirectory(sPath);
	}

	char sPathComplete[PLATFORM_MAX_PATH];
	GetConVarString(gbStvPathComplete, sPathComplete, sizeof sPathComplete);
	if(!DirExists(sPathComplete))
	{
		initDirectory(sPathComplete);
	}

	CreateTimer(300.0, Timer_CheckStatus, _, TIMER_REPEAT);

	StopRecord();
}


public void OnClientDisconnect_Post(int client)
{
	CheckStatus();
}


public void OnMapEnd()
{
	if(gIsRecording)
	{
		StopRecord();
		gIsManual = false;
	}
}


public void OnMapStart()
{
	authenticateServer();
}


public APLRes AskPluginLoad2(Handle myself, bool late, char[] error, int err_max)
{
	CreateNative("GB_BanClient", Native_GB_BanClient);

	gLateLoaded = late;

	return APLRes_Success;
}
