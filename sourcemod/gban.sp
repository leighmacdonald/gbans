#pragma semicolon 1
#pragma tabsize 4
#define DEBUG

#define PLUGIN_AUTHOR "Leigh MacDonald"
#define PLUGIN_VERSION "0.00"
#define PLUGIN_NAME "gban"

#include <sourcemod>
#include <sdktools>
#include <tf2>
#include <tf2_stocks>
#include <ripext>
//#include <sdkhooks>

HTTPClient httpClient; 

bool PlayerAllowed[MAXPLAYERS+1];

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
	//RegAdminCmd("gb_ban", CommandBan);
	CreateConVar("sb_version", PLUGIN_VERSION, _, FCVAR_SPONLY | FCVAR_REPLICATED | FCVAR_NOTIFY);
	PrintToServer("Hi");
	httpClient = new HTTPClient("http://172.16.1.22:6969");
	JSONObject authReq = new JSONObject(); 
	authReq.SetString("server_id", "af-1");
	authReq.SetString("key", "opensesame");
	httpClient.Post("v1/auth", authReq, OnAuthReqReceived); 
	delete authReq;
}

void OnAuthReqReceived(HTTPResponse response, any value)
{
    if (response.Status != HTTPStatus_OK) {
        return;
    }
    if (response.Data == null) {
        // Invalid JSON response
        return;
    }

    JSONObject authResp = view_as<JSONObject>(response.Data);
    bool status = authResp.GetBool("status");
	char buff[20];
	authResp.GetString("token", buff, 20);
    PrintToServer("%d %s", status, buff);
}  

public Action CommandBan(int client, int args) {
	PrintToServer("Ban");
}


public bool OnClientConnect(int client, char[] rejectmsg, int maxlen)
{
	PlayerAllowed[client] = false;
	return true;
}

public void OnClientAuthorized(int client, const char[] auth)
{
	/* Do not check bots nor check player with lan steamid. */
	if (auth[0] == 'B' || auth[9] == 'L')
	{
		//PlayerStatus[client] = true;
		return;
	}

	char ip[30];


	GetClientIP(client, ip, sizeof(ip));

	#if defined DEBUG
	PrintToServer("Checking ban for: %s", auth);
	#endif

	//DB.Query(VerifyBan, Query, GetClientUserId(client), DBPrio_High);
}