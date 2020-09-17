#pragma semicolon 1

#define DEBUG

#define PLUGIN_AUTHOR "Leigh MacDonald"
#define PLUGIN_VERSION "0.00"
#defind PLUGIN_NAME "gban"

#include <sourcemod>
#include <sdktools>
#include <tf2>
#include <tf2_stocks>
#include <httpreq>
//#include <sdkhooks>

public Plugin myinfo = 
{
	name = PLUGIN_NAME,
	author = PLUGIN_AUTHOR,
	description = "gban game client",
	version = PLUGIN_VERSION,
	url = "https://github.com/leighmacdonald/gban"
};

public void OnPluginStart()
{
	RegAdminCmd("gb_ban", CommandBan, ADMINFLAG_SLAY)
	PrintToServer("Hi")
}

public Action CommandBan(int client, int args) {

}


public bool OnClientConnect(int client, char[] rejectmsg, int maxlen)
{
	PlayerStatus[client] = false;
	return true;
}

public void OnClientAuthorized(int client, const char[] auth)
{
	/* Do not check bots nor check player with lan steamid. */
	if (auth[0] == 'B' || auth[9] == 'L' || DB == INVALID_HANDLE)
	{
		PlayerStatus[client] = true;
		return;
	}

	char Query[256], ip[30];

	GetClientIP(client, ip, sizeof(ip));

	FormatEx(Query, sizeof(Query), "SELECT bid, ip FROM %s_bans WHERE ((type = 0 AND authid REGEXP '^STEAM_[0-9]:%s$') OR (type = 1 AND ip = '%s')) AND (length = '0' OR ends > UNIX_TIMESTAMP()) AND RemoveType IS NULL", DatabasePrefix, auth[8], ip);

	#if defined DEBUG
	LogToFile(logFile, "Checking ban for: %s", auth);
	#endif

	DB.Query(VerifyBan, Query, GetClientUserId(client), DBPrio_High);
}