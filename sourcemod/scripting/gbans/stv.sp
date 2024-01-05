/*
 * Adapted from: Auto Recorder http://forums.alliedmods.net/showthread.php?t=92072
 */
#pragma semicolon 1
#pragma tabsize 4
#pragma newdecls required

#include <json>
#include <sourcemod>
#include <sourcetvmanager>
#include <tf2_stocks>
#include "globals.sp"

public void onPluginStartSTV()
{
	// STV settings
	gAutoRecord = CreateConVar("gb_stv_enable", "1", "Enable automatic recording", _, true, 0.0, true, 1.0);
	gMinPlayersStart = CreateConVar("gb_stv_minplayers", "1", "Minimum players on server to start recording", _, true, 0.0);
	gIgnoreBots = CreateConVar("gb_stv_ignorebots", "1", "Ignore bots in the player count", _, true, 0.0, true, 1.0);
	gTimeStart = CreateConVar("gb_stv_timestart", "-1", "Hour in the day to start recording (0-23, -1 disables)");
	gTimeStop = CreateConVar("gb_stv_timestop", "-1", "Hour in the day to stop recording (0-23, -1 disables)");
	gFinishMap = CreateConVar("gb_stv_finishmap", "1", "If 1, continue recording until the map ends", _, true, 0.0, true, 1.0);
	gDemoPathActive = CreateConVar("gb_stv_path", "stv_demos/active", "Path to store currently recording demos");
	gDemoPathComplete = CreateConVar("gb_stv_path_complete", "stv_demos/complete", "Path to store complete demos");
}


public void setupSTV()
{
	RegAdminCmd("gb_stv_record", Command_Record, ADMFLAG_KICK, "Starts a SourceTV demo");
	RegAdminCmd("gb_stv_stoprecord", Command_StopRecord, ADMFLAG_KICK, "Stops the current SourceTV demo");

	gTvEnabled = FindConVar("tv_enable");
	char sPath[PLATFORM_MAX_PATH];
	gDemoPathActive.GetString(sPath, sizeof sPath);
	if(!DirExists(sPath))
	{
		initDirectory(sPath);
	}

	char sPathComplete[PLATFORM_MAX_PATH];
	gDemoPathComplete.GetString(sPathComplete, sizeof sPathComplete);
	if(!DirExists(sPathComplete))
	{
		initDirectory(sPathComplete);
	}

	gMinPlayersStart.AddChangeHook(OnConVarChanged);
	gIgnoreBots.AddChangeHook(OnConVarChanged);
	gTimeStart.AddChangeHook(OnConVarChanged);
	gTimeStop.AddChangeHook(OnConVarChanged);
	gDemoPathActive.AddChangeHook(OnConVarChanged);

	CreateTimer(300.0, Timer_CheckStatus, _, TIMER_REPEAT);

	StopRecord();
	CheckStatus();
}


public void OnMapStart()
{
	reloadAdmins();
	if(!gStvMapChanged)
	{
	// STV does not function until a map change has occurred.
		gbLog("Restarting map to enabled STV");
		gStvMapChanged = true;
		char mapName[128];
		GetCurrentMap(mapName, sizeof mapName);
		ForceChangeLevel(mapName, "Enable STV");
	}
}


public void OnConVarChanged(ConVar convar, const char[] oldValue, const char[] newValue)
{
	if(convar == gDemoPathActive || convar == gDemoPathComplete)
	{
		if(!DirExists(newValue))
		{
			initDirectory(newValue);
		}
	}
	else
	{
		CheckStatus();
	}
}


public void onMapEndSTV()
{
	if(gIsRecording)
	{
		StopRecord();
		gIsManual = false;
	}
}


public void OnClientPutInServerSTV(int client)
{
	CheckStatus();
}


public void OnClientDisconnect_Post(int client)
{
	CheckStatus();
}


public Action Timer_CheckStatus(Handle timer)
{
	CheckStatus();
	return Plugin_Handled;
}


public Action Command_Record(int client, int args)
{
	if(gIsRecording)
	{
		ReplyToCommand(client, "[GB] SourceTV is already recording!");
		return Plugin_Handled;
	}

	StartRecord();
	gIsManual = true;
	ReplyToCommand(client, "[GB] SourceTV is now recording...");
	return Plugin_Handled;
}


public Action Command_StopRecord(int client, int args)
{
	if(!gIsRecording)
	{
		ReplyToCommand(client, "[GB] SourceTV is not recording!");
		return Plugin_Handled;
	}
	StopRecord();
	if(gIsManual)
	{
		gIsManual = false;
		CheckStatus();
	}
	ReplyToCommand(client, "[GB] Stopped recording.");
	return Plugin_Handled;
}


void CheckStatus()
{
	if(gAutoRecord.BoolValue && !gIsManual)
	{
		int iTimeStart = gTimeStart.IntValue;
		int iTimeStop = gTimeStop.IntValue;
		bool bReverseTimes = (iTimeStart > iTimeStop);
		char sCurrentTime[4];
		FormatTime(sCurrentTime, sizeof sCurrentTime, "%H", GetTime());
		int iCurrentTime = StringToInt(sCurrentTime);
		if(GetPlayerCount() >= gMinPlayersStart.IntValue && (iTimeStart < 0 || (iCurrentTime >= iTimeStart && (bReverseTimes || iCurrentTime < iTimeStop))))
		{
			StartRecord();
		}
		else if(gIsRecording && !gFinishMap.BoolValue && (iTimeStop < 0 || iCurrentTime >= iTimeStop))
		{
			StopRecord();
		}
	}
}


int GetPlayerCount()
{
	bool bIgnoreBots = gIgnoreBots.BoolValue;

	int iNumPlayers = 0;
	for(int i = 1; i <= MaxClients; i++)
	{
		if(IsClientConnected(i) && (!bIgnoreBots || !IsFakeClient(i)))
		{
			iNumPlayers++;
		}
	}

	if(!bIgnoreBots)
	{
		iNumPlayers--;
	}

	return iNumPlayers;
}


void StartRecord()
{
	if(gTvEnabled.BoolValue && !gIsRecording)
	{
		char sPath[PLATFORM_MAX_PATH];
		char sTime[16];
		char sMap[64];

		gDemoPathActive.GetString(sPath, sizeof sPath);
		FormatTime(sTime, sizeof sTime, "%Y%m%d-%H%M%S", GetTime());
		GetCurrentMap(sMap, sizeof sMap);

		// replace slashes in map path name with dashes, to prevent fail on workshop maps
		ReplaceString(sMap, sizeof sMap, "/", "-", false);
		ReplaceString(sMap, sizeof sMap, ".", "-", false);

		ServerCommand("tv_record \"%s/%s-%s\"", sPath, sTime, sMap);
		gIsRecording = true;

		gbLog("Recording to %s-%s.dem", sTime, sMap);
	}
}


void StopRecord()
{
	if(gTvEnabled.BoolValue)
	{
		ServerCommand("tv_stoprecord");
		gIsRecording = false;
	}
}

public void SourceTV_OnStopRecording(int instance, const char[] filename, int recordingtick)
{
	char sPieces[32][PLATFORM_MAX_PATH];
	char outPath[PLATFORM_MAX_PATH];

	gDemoPathComplete.GetString(outPath, sizeof outPath);

	int iNumPieces = ExplodeString(filename, "/", sPieces, sizeof sPieces, sizeof sPieces[] );

	Format(outPath, sizeof outPath, "%s/%s", outPath, sPieces[iNumPieces - 1]);

	gbLog("Writing stv: %s dest: %s", filename, outPath);
	if(!RenameFile(outPath, filename))
	{
		gbLog("Failed to rename completed demo file");
		return ;
	}
	gbLog("Wrote demo");
}


void initDirectory(const char[] sDir)
{
	char sPieces[32][PLATFORM_MAX_PATH];
	char sPath[PLATFORM_MAX_PATH];
	int iNumPieces = ExplodeString(sDir, "/", sPieces, sizeof sPieces, sizeof sPieces[] );

	for(int i = 0; i < iNumPieces; i++)
	{
		Format(sPath, sizeof sPath, "%s/%s", sPath, sPieces[i]);
		if(!DirExists(sPath))
		{
			CreateDirectory(sPath, 509);
		}
	}
}
