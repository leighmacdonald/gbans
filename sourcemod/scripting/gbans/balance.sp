#pragma semicolon 1
#pragma tabsize 4
#pragma newdecls required

public Action onCmdAutoTeamAction(int clientId, int argc)
{
    if (gDisableAutoTeam.BoolValue) {
        return Plugin_Handled;
    }

	return Plugin_Continue;
}