#pragma semicolon 1
#pragma tabsize 4
#pragma newdecls required

#include "globals.sp"
#include "common.sp"
#include "ripext/json"
#include "ripext/http"

public Action onCmdVersion(int clientId, int args)
{
	ReplyToCommand(clientId, "[GB] Version %s", PLUGIN_VERSION);
	return Plugin_Handled;
}

/**
Ping the moderators through discord
*/
public Action onCmdMod(int clientId, int argc)
{
	if(argc < 1)
	{
		ReplyToCommand(clientId, "Must supply a reason message for pinging");
		return Plugin_Handled;
	}
	char reason[256];
	for(int i = 1; i <= argc; i++)
	{
		if(i > 1)
		{
			StrCat(reason, sizeof reason, " ");
		}
		char buff[128];
		GetCmdArg(i, buff, sizeof buff);
		StrCat(reason, sizeof reason, buff);
	}
	char authId[50];
	if(!GetClientAuthId(clientId, AuthId_Steam3, authId, sizeof authId, true))
	{
		ReplyToCommand(clientId, "Failed to get auth_id of user: %d", clientId);
		return Plugin_Continue;
	}
	char name[64];
	if(!GetClientName(clientId, name, sizeof name))
	{
		gbLog("Failed to get user name?");
		return Plugin_Continue;
	}

	char serverName[PLATFORM_MAX_PATH];
	GetConVarString(gbCoreHost, serverName, sizeof serverName);

	JSONObject obj = new JSONObject();
	obj.SetString("steamId", authId);
	obj.SetString("name", name);
	obj.SetString("reason", reason);
	obj.SetInt("client", clientId);

	postHTTPRequest("/connect/sourcemod.v1.PluginService/SMPingMod", obj, onPingModRespReceived);

	return Plugin_Handled;
}

void onPingModRespReceived(HTTPResponse response, any clientId) {
	if (response.Status != HTTPStatus_OK) {
		LogError("Invalid report response code: %d", response.Status);

        return;
    } 
	ReplyToCommand(clientId, "Mods have been alerted, thanks!");
}

public Action onCmdSeed(int clientId, int argc) {
	char authId[50];
	if(!GetClientAuthId(clientId, AuthId_Steam3, authId, sizeof authId, true))
	{
		ReplyToCommand(clientId, "Failed to get auth_id of user: %d", clientId);
		return Plugin_Continue;
	}

	JSONObject obj = new JSONObject();
	obj.SetString("steamId", authId);
	obj.SetInt("clientId", clientId);

	postHTTPRequest("/connect/sourcemod.v1.PluginService/SMSeed", obj, onCmdSeedReceived);

	return Plugin_Handled;
}

void onCmdSeedReceived(HTTPResponse response, any clientId) {
	switch (response.Status) {
		case HTTPStatus_TooManyRequests: {
			ReplyToCommand(clientId, "Please wait before making new seed requests (5min cooldown)");
			return;
		}
		case HTTPStatus_OK: {
			ReplyToCommand(clientId, "Mods have been alerted, thanks!");
			return;
		}
		default: {
			ReplyToCommand(clientId, "Got invalid response code :(");
			return;
		}
	}
}

public Action onCmdHelp(int clientId, int argc)
{
	onCmdVersion(clientId, argc);
	ReplyToCommand(clientId, "gb_ban #user duration [reason]");
	ReplyToCommand(clientId, "gb_ban_ip #user duration [reason]");
	ReplyToCommand(clientId, "gb_kick #user [reason]");
	ReplyToCommand(clientId, "gb_mute #user duration [reason]");
	ReplyToCommand(clientId, "gb_mod reason");
	ReplyToCommand(clientId, "gb_version -- Show the current version");
	return Plugin_Handled;
}

public Action onAdminCmdBan(int clientId, int argc)
{
	char command[64];
	char targetIdStr[50];
	char memo[256];
	GB_BanReason reason;
	int duration;
	int bantype;

	char usage[] = "Usage: %s <targetId> <reason> <duration> <bantype> <memo>";

	GetCmdArg(0, command, sizeof command);

	if(argc < 4)
	{
		char usageReply[256];
		Format(usageReply, sizeof usageReply, usage, command);
		reply(clientId, usageReply);
		return Plugin_Handled;
	}

	GetCmdArg(1, targetIdStr, sizeof targetIdStr);

	int reasonInt = 0;
	if (!GetCmdArgIntEx(2, reasonInt)) {
		reply(clientId, "Failed to parse reason");
		return Plugin_Handled;
	}

	if(reasonInt < view_as<int>(custom) || reasonInt > view_as<int>(itemDescriptions))
	{
		reply(clientId, "Invalid reason value. Out of range.");
		return Plugin_Handled;
	}

	reason = view_as<GB_BanReason>(reasonInt);

	if (!GetCmdArgIntEx(3, duration)) {
		reply(clientId, "Failed to parse duration");
		return Plugin_Handled;
	}

	if (!GetCmdArgIntEx(4, bantype)) {
		reply(clientId, "Failed to parse bantype");
		return Plugin_Handled;
	}

	gbLog("args: %d", argc);
	if (argc > 4) {
		GetCmdArg(5, memo, sizeof memo);
	} else {
		Format(memo, sizeof memo, "in-game ban");
	}
	
	gbLog("Target: %s reason: %d duration: %d bantype: %d memo: %s", targetIdStr, reason, duration, bantype, memo);

	int targetIdx = FindTarget(clientId, targetIdStr, true, false);
	if(targetIdx < 0)
	{
		reply(clientId, "Failed to locate user");
		return Plugin_Handled;
	}


	if(!ban(clientId, targetIdx, reason, duration, bantype, memo))
	{
		reply(clientId, "Error sending ban request");
	}

	return Plugin_Handled;
}
