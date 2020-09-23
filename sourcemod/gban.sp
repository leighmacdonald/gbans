#pragma semicolon 1
#pragma tabsize 4

#include <sourcemod>
#include <sdktools>
#include <tf2>
#include <tf2_stocks>
#include <system2>
#include <json>

#define DEBUG

#define PLUGIN_AUTHOR "Leigh MacDonald"
#define PLUGIN_VERSION "0.00"
#define PLUGIN_NAME "gban"

// Ban states retured from server
#define BSUnknown -1
#define BSOK 0
#define BSNoComm 1
#define BSBanned 2

// Authentication token len
#define TOKEN_LEN 40

enum struct PlayerInfo {
	bool authed;
	char ip[16];
	int ban_type;
}

// Globals must all start with g_
char g_token[TOKEN_LEN+1]; // tokens are 40 chars + term

PlayerInfo g_players[MAXPLAYERS+1];

ConVar g_gb_host;
ConVar g_gb_port;
ConVar g_gb_server_name;
ConVar g_gb_key;

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
	ReadConfig();
	AuthenticateServer();
}

void ReadConfig() {
	g_gb_host = CreateConVar("gb_host", "http://172.16.1.22", "Remote gban server host");
	g_gb_port = CreateConVar("gb_port", "6969", "Remote gban server port");
	g_gb_server_name = CreateConVar("gb_server_name", "default", "Unique server name for this server");
	g_gb_key = CreateConVar("gb_key", "test_auth", "The authentication key used to retrieve a auth token");
	RegConsoleCmd("gb_version", Command_Version, "Get gban version");
}


System2HTTPRequest newReq(System2HTTPResponseCallback cb, const char[] path) {
	decl String:addr[256];
	g_gb_host.GetString(addr, sizeof(addr));
	decl String:fullAddr[1024];
	Format(fullAddr, sizeof(fullAddr), "%s%s", addr, path);
	int port = g_gb_port.IntValue;
	System2HTTPRequest httpRequest = new System2HTTPRequest(cb, fullAddr); 
	httpRequest.SetPort(port);
	httpRequest.SetHeader("Content-Type", "application/json"); 
	if (strlen(g_token) == TOKEN_LEN) {
		httpRequest.SetHeader("Authorization", g_token);
	}
	return httpRequest;
}

void CheckPlayer(int client, const char[] auth, const char[] ip) {
	JSON_Object obj = new JSON_Object();
	obj.SetString("steam_id", auth);
	obj.SetInt("client_id", client);
	obj.SetString("ip", ip);
	char encoded[1024];
	obj.Encode(encoded, sizeof(encoded));
	obj.Cleanup();

	System2HTTPRequest req = newReq(OnCheckResp, "/v1/check");
	req.SetData(encoded);
	req.POST();
	delete req;  
}

void OnCheckResp(bool success, const char[] error, System2HTTPRequest request, System2HTTPResponse response, HTTPRequestMethod method)
{
    if (success) {
        char lastURL[128];
        response.GetLastURL(lastURL, sizeof(lastURL));
        int statusCode = response.StatusCode;
        float totalTime = response.TotalTime;
		#if defined DEBUG
        PrintToServer("[GB] Request to %s finished with status code %d in %.2f seconds", lastURL, statusCode, totalTime);
		#endif
		char[] content = new char[response.ContentLength + 1];
        response.GetContent(content, response.ContentLength + 1); 

		if (statusCode != 200) {
			// Fail open if the server is broken
			return;
		}
		JSON_Object resp = json_decode(content);
		bool client_id = resp.GetBool("client_id");
		int ban_type = resp.GetInt("ban_type");
		char msg[256];
		resp.GetString("msg", msg, sizeof(msg));
		PrintToServer("[GB] Ban state: %d", ban_type);
		if (ban_type >= BSBanned) {
			KickClient(client_id, msg);
			return;
		}
		char ip[16];
		GetClientIP(client_id, ip, sizeof(ip));
		g_players[client_id].authed = true;
		g_players[client_id].ip = ip;
		g_players[client_id].ban_type = ban_type;
		PrintToServer("[GB] Successfully authenticated with gban server");
    } else {
        PrintToServer("[GB] Error on authentication request: %s", error);
    } 
}  

void AuthenticateServer() {
	decl String:server_name[40];
	decl String:key[40];
	g_gb_server_name.GetString(server_name, sizeof(server_name));
	g_gb_key.GetString(key, sizeof(key));
	JSON_Object obj = new JSON_Object();
	obj.SetString("server_name", server_name);
	obj.SetString("key", key);
	char encoded[1024];
	obj.Encode(encoded, sizeof(encoded));
	obj.Cleanup();
	delete obj;
	System2HTTPRequest req = newReq(OnAuthReqReceived, "/v1/auth");
	req.SetData(encoded);
	req.POST();
	delete req;  
}

void OnAuthReqReceived(bool success, const char[] error, System2HTTPRequest request, System2HTTPResponse response, HTTPRequestMethod method)
{
    if (success) {
        char lastURL[128];
        response.GetLastURL(lastURL, sizeof(lastURL));

        int statusCode = response.StatusCode;
        float totalTime = response.TotalTime;
		#if defined DEBUG
        PrintToServer("[GB] Request to %s finished with status code %d in %.2f seconds", lastURL, statusCode, totalTime);
		#endif
		char[] content = new char[response.ContentLength + 1];
        response.GetContent(content, response.ContentLength + 1); 
		JSON_Object resp = json_decode(content);
		bool ok = resp.GetBool("status");
		if (!ok) {
			PrintToServer("[GB] Invalid response status, cannot authenticate");
			return;
		}
		decl String:token[41];
		resp.GetString("token", token, sizeof(token));
		if (strlen(token) != 40) {
			PrintToServer("[GB] Invalid response status, invalid token");
			return;
		}
		g_token = token;
		PrintToServer("[GB] Successfully authenticated with gban server");
    } else {
        PrintToServer("[GB] Error on authentication request: %s", error);
    } 
}  

public void SendDiscord(char[] body) {
}

public Action CommandBan(int client, int args) {
	PrintToServer("Ban");
}

public Action Command_Version(int client, int args)
{
	ReplyToCommand(client, "[GB] Version %s", PLUGIN_VERSION);
	return Plugin_Handled;
}

public bool OnClientConnect(int client, char[] rejectmsg, int maxlen)
{
	g_players[client].authed = false;
	g_players[client].ban_type = BSUnknown;
	return true;
}

public void OnClientAuthorized(int client, const char[] auth)
{
	char ip[16];
	GetClientIP(client, ip, sizeof(ip));
	GetClientUserId(client);
	/* Do not check bots nor check player with lan steamid. */
	if (auth[0] == 'B' /*|| auth[9] == 'L'*/ )
	{
		g_players[client].authed = true;
		g_players[client].ip = ip;
		return;
	}

	#if defined DEBUG
	PrintToServer("Checking ban for: %s", auth);
	#endif

	CheckPlayer(client, auth, ip);
}
