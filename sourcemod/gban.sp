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

HTTPClient g_httpClient; 

char g_token[40];
ConVar g_gb_host;
ConVar g_gb_server_name;
ConVar g_gb_version;
ConVar g_gb_key;

enum struct PlayerInfo {
	bool authed;
	char address[25];
}

PlayerInfo g_players[MAXPLAYERS+1];

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
	InitHTTP();
	Authenticate();
}

void ReadConfig() {
	g_gb_version = CreateConVar("gb_version", PLUGIN_VERSION, _, FCVAR_SPONLY | FCVAR_REPLICATED | FCVAR_NOTIFY);
	g_gb_host = CreateConVar("gb_host", "http://172.16.1.22:6969", "Remote gban server host");
	g_gb_server_name = CreateConVar("gb_server_name", "Default", "Unique server name for this server");
	g_gb_key = CreateConVar("gb_key", "empty", "The authentication key used to retrieve a auth token");
}

void InitHTTP() {	
	char host[256];
	g_gb_host.GetString(host, sizeof(host));
	g_httpClient = new HTTPClient(host);
	//g_httpClient.SetHeader("Transfer-Encoding", "identity");
}

void Authenticate() {
	decl String:server_name[40];
	decl String:key[40];
	g_gb_server_name.GetString(server_name, sizeof(server_name));
	g_gb_key.GetString(key, sizeof(key));
	JSONObject authReq = new JSONObject();
	authReq.SetString("server_name", server_name);
	authReq.SetString("key", key);
	g_httpClient.Post("v1/auth", authReq, OnAuthReqReceived); 
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
	char buff[40];
	authResp.GetString("token", buff, strlen(buff));
    PrintToServer("%d %s", status, buff);
}  

public void SendDiscord(char[] body) {
}

public Action CommandBan(int client, int args) {
	PrintToServer("Ban");
}

public bool OnClientConnect(int client, char[] rejectmsg, int maxlen)
{
	g_players[client].authed = false;
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
	GetClientUserId(client);

	#if defined DEBUG
	PrintToServer("Checking ban for: %s", auth);
	#endif

}


methodmap AuthReq < JSONObject
{
    // Constructor
    public AuthReq() { return view_as<AuthReq>(new JSONObject()); }

    public void GetServerName(char[] buffer, int maxlength)
    {
        this.GetString("server_name", buffer, maxlength);
    }
    public void SetServerName(const char[] value)
    {
        this.SetString("server_name", value);
    }
    public void GetKey(char[] buffer, int maxlength)
    {
        this.GetString("key", buffer, maxlength);
    }
    public void SetKey(const char[] value)
    {
        this.SetString("key", value);
    }
};  