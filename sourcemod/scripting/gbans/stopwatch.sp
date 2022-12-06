
public onPluginStartStopwatch() {
	HookEvent("teamplay_round_start", onTeamplayRoundStart, EventHookMode_PostNoCopy);
	RegConsoleCmd("tournament_readystate", cmd_block);
	RegConsoleCmd("tournament_teamname", cmd_block);
}

public onMapEndStopwatch() {
    if (GetConVarBool(FindConVar("mp_tournament"))) {
	    SetConVarBool(FindConVar("mp_tournament"), false);
    }
}

public onTeamplayRoundStart(Handle:event, const String:name[], bool:dontBroadcast) {
	if (!gStopwatchEnabled.BoolValue || GetConVarBool(FindConVar("mp_tournament"))) {
        return 
    }
    // set cvars
    SetConVarBool(FindConVar("mp_tournament"), true);
    SetConVarBool(FindConVar("mp_tournament_allow_non_admin_restart"), false);
    SetConVarBool(FindConVar("mp_tournament_stopwatch"), true);

    // set team names
    char teamnameA[16];
    gStopwatchNameBlu.GetString(teamnameA, sizeof(teamnameA));
    SetConVarString(FindConVar("mp_tournament_blueteamname"), teamnameA);

    char teamnameB[16];
    gStopwatchNameRed.GetString(teamnameB, sizeof(teamnameB));
    SetConVarString(FindConVar("mp_tournament_redteamname"), teamnameB);
    
    // wait for players, then start the tournament
    ServerCommand("mp_restartgame %d", GetConVarInt(FindConVar("mp_waitingforplayers_time")));

    AllowMatch();
	
}

stock void AllowMatch()
{
    for(int i = 1; i <= MaxClients; i++)
    {
        GameRules_SetProp("m_bTeamReady", 1, .element=i);
    }
}

public Action:cmd_block(client, args) {
	return (gStopwatchEnabled.BoolValue ? Plugin_Handled : Plugin_Continue);
}
