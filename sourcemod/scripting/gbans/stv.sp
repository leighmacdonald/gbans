/*
 * Adapted from: Auto Recorder http://forums.alliedmods.net/showthread.php?t=92072
 */
#pragma semicolon 1
#pragma tabsize 4
#pragma newdecls required

public
void OnConVarChanged(ConVar convar, const char[] oldValue, const char[] newValue) {
    if (convar == gb_stv_path || convar == gb_stv_path_complete) {
        if (!DirExists(newValue)) {
            initDirectory(newValue);
        }
    } else {
        CheckStatus();
    }
}

public
Action Timer_CheckStatus(Handle timer) {
    CheckStatus();
    return Plugin_Handled;
}

public
Action Command_Record(int client, int args) {
    if (gIsRecording) {
        ReplyToCommand(client, "[GB] SourceTV is already recording!");
        return Plugin_Handled;
    }

    StartRecord();
    gIsManual = true;
    ReplyToCommand(client, "[GB] SourceTV is now recording...");
    return Plugin_Handled;
}

public
Action Command_StopRecord(int client, int args) {
    if (!gIsRecording) {
        ReplyToCommand(client, "[GB] SourceTV is not recording!");
        return Plugin_Handled;
    }

    StopRecord();

    if (gIsManual) {
        gIsManual = false;
        CheckStatus();
    }

    ReplyToCommand(client, "[GB] Stopped recording.");

    return Plugin_Handled;
}

void CheckStatus() {
    if (GetConVarBool(gb_auto_record) && !gIsManual) {
        int  iTimeStart    = GetConVarInt(gb_stv_timestart);
        int  iTimeStop     = GetConVarInt(gb_stv_timestop);
        bool bReverseTimes = (iTimeStart > iTimeStop);
        char sCurrentTime[4];
        FormatTime(sCurrentTime, sizeof sCurrentTime, "%H", GetTime());
        int iCurrentTime = StringToInt(sCurrentTime);
        if (GetPlayerCount() >= GetConVarInt(gb_stv_minplayers)
            && (iTimeStart < 0 || (iCurrentTime >= iTimeStart && (bReverseTimes || iCurrentTime < iTimeStop)))) {
            StartRecord();
        } else if (gIsRecording && !GetConVarBool(gb_stv_finishmap) && (iTimeStop < 0 || iCurrentTime >= iTimeStop)) {
            StopRecord();
        }
    }
}

int GetPlayerCount() {
    bool bIgnoreBots = GetConVarBool(gb_stv_ignorebots);

    int iNumPlayers = 0;
    for (int i = 1; i <= MaxClients; i++) {
        if (IsClientConnected(i) && (!bIgnoreBots || !IsFakeClient(i))) {
            iNumPlayers++;
        }
    }

    if (!bIgnoreBots) {
        iNumPlayers--;
    }

    return iNumPlayers;
}

void StartRecord() {
    if (GetConVarBool(gb_stv_enable) && !gIsRecording) {
        char sPath[PLATFORM_MAX_PATH];
        char sTime[16];
        char sMap[64];

        gb_stv_path.GetString(sPath, sizeof sPath);
        FormatTime(sTime, sizeof sTime, "%Y%m%d-%H%M%S", GetTime());
        GetCurrentMap(sMap, sizeof sMap);

        // replace slashes in map path name with dashes, to prevent fail on workshop maps
        ReplaceString(sMap, sizeof sMap, "/", "-", false);
        ReplaceString(sMap, sizeof sMap, ".", "-", false);

        ServerCommand("tv_record \"%s/%s-%s\"", sPath, sTime, sMap);
        gIsRecording = true;

        LogMessage("Recording to %s-%s.dem", sTime, sMap);
    }
}

void StopRecord() {
    if (!gIsRecording) {
        return;
    }

    gIsRecording = false;

    // gb_stv_enable only gates starting a recording. Always stop one that is
    // already running, otherwise disabling the convar mid-recording leaves the
    // flag set forever and auto-recording never restarts.
    ServerCommand("tv_stoprecord");
}

public
void SourceTV_OnStopRecording(int instance, const char[] filename, int recordingtick) {
    char sPieces[32][PLATFORM_MAX_PATH];
    char outPath[PLATFORM_MAX_PATH];

    GetConVarString(gb_stv_path_complete, outPath, sizeof outPath);

    int iNumPieces = ExplodeString(filename, "/", sPieces, sizeof sPieces, sizeof sPieces[]);
    if (iNumPieces < 1) {
        LogError("Failed to parse completed demo filename \"%s\"", filename);

        return;
    }

    Format(outPath, sizeof outPath, "%s/%s", outPath, sPieces[iNumPieces - 1]);

    LogMessage("Writing stv: %s dest: %s", filename, outPath);
    if (!RenameFile(outPath, filename)) {
        LogError("Failed to rename completed demo file");
        return;
    }
}

/**
 * Only allow SourceMod admins with kick privileges to watch SourceTV. This
 * runs once the spectator is in game so their SteamID is available for the
 * admin lookup; unauthorized spectators are kicked immediately.
 * 
 * This exists because sometimes on slower systems the first user connecting 
 * when coming out of hiberation can be kicked due to net_chan_limit_msec
 * getting triggered while doing the warmup phase.
 */
public
void SourceTV_OnSpectatorPutInServer(int client) {
    if (!SourceTV_IsClientConnected(client) || SourceTV_IsClientProxy(client)) {
        return;
    }

    AdminId admin = INVALID_ADMIN_ID;

    char steam2[32];
    if (GetClientAuthId(client, AuthId_Steam2, steam2, sizeof steam2)) {
        admin = FindAdminByIdentity(AUTHMETHOD_STEAM, steam2);
    }

    if (admin == INVALID_ADMIN_ID) {
        char steam64[32];
        if (GetClientAuthId(client, AuthId_SteamID64, steam64, sizeof steam64)) {
            admin = FindAdminByIdentity(AUTHMETHOD_STEAM, steam64);
        }
    }

    if (admin != INVALID_ADMIN_ID && GetAdminFlag(admin, Admin_Kick, Access_Effective)) {
        return;
    }

    char name[64];
    SourceTV_GetClientName(client, name, sizeof name);
    LogMessage("Kicking SourceTV spectator %s: missing kick privilege", name);
    SourceTV_KickClient(client, "SourceTV access requires the SourceMod kick privilege (ADMFLAG_KICK)");
}

/**
 * Relay chat messages typed by SourceTV spectators to the in-game players so
 * the two audiences can talk to each other.
 */
public
void SourceTV_OnSpectatorChatMessage_Post(int client, const char[] message, const char[] chatgroup) {
    if (!GetConVarBool(gb_stv_chat_forward) || message[0] == '\0') {
        return;
    }

    if (!SourceTV_IsClientConnected(client) || SourceTV_IsClientProxy(client)) {
        return;
    }

    char name[64];
    SourceTV_GetClientName(client, name, sizeof name);
    if (name[0] == '\0') {
        strcopy(name, sizeof name, "SourceTV");
    }

    PrintToChatAll("[STV] %s: %s", name, message);
    LogMessage("STV chat [%s] %s: %s", chatgroup, name, message);
}

void initDirectory(const char[] sDir) {
    char sPieces[32][PLATFORM_MAX_PATH];
    char sPath[PLATFORM_MAX_PATH];
    char sNext[PLATFORM_MAX_PATH];
    int  iNumPieces = ExplodeString(sDir, "/", sPieces, sizeof sPieces, sizeof sPieces[]);

    for (int i = 0; i < iNumPieces; i++) {
        Format(sNext, sizeof sNext, "%s/%s", sPath, sPieces[i]);
        strcopy(sPath, sizeof sPath, sNext);

        if (!DirExists(sPath)) {
            CreateDirectory(sPath, 509);
        }
    }
}
