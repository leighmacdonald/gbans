#pragma semicolon 1
#pragma tabsize 4
#pragma newdecls required

#include "gbans"

const TOKEN_LEN = 1024;

void PrintRPCError(HTTPResponse response) {
    char code[64];
    char message[1024];

    LogError("Invalid response code: %d", response.Status);
    JSONObject err_obj = view_as<JSONObject>(response.Data);
    if (!err_obj.GetString("code", code, sizeof code)) {
        return;
    }
    if (!err_obj.GetString("message", message, sizeof code)) {
        return;
    }

	LogError("ConnectRPC Error code=%s message=%s", code, message);
}

public bool parseReason(const char[] reasonStr, GB_BanReason &reason)
{
	int reasonInt = StringToInt(reasonStr, 10);
	if(reasonInt <= 0 || reasonInt < view_as<int>(custom) || reasonInt > view_as<int>(itemDescriptions))
	{
		return false;
	}
	reason = view_as<GB_BanReason>(reasonInt);
	return true;
}

void postHTTPRequest(const char[] path, JSON data, HTTPRequestCallback callback, any value = 0) {
	char url[1024];
	makeURL(path, url, sizeof url);

	HTTPRequest request = new HTTPRequest(url);
	request.SetHeader("Content-Type", "application/json");
	if (gToken[0] != '\0') {
	    char authHeader[1024] = "Bearer ";
		StrCat(authHeader, sizeof authHeader, gToken);
		request.SetHeader("Authorization", authHeader);
	}

    request.Post(data, callback, value);

	CloseHandle(data);
}

stock void makeURL(const char[] path, char[] outURL, int maxLen) {
	char serverHost[PLATFORM_MAX_PATH];
	GetConVarString(gbCoreHost, serverHost, sizeof serverHost);
	int port = GetConVarInt(gbCorePort);

	Format(outURL, maxLen, "%s:%d%s", serverHost, port, path);
}

stock bool isValidClient(int client)
{
	if(!(1 <= client <= MaxClients) || !IsClientInGame(client) || IsFakeClient(client) || IsClientSourceTV(client) || IsClientReplay(client))
	{
		return false;
	}
	return true;
}

stock int GetRealClientCount()
{
	int iClients = 0;
	for(int i = 1; i <= MaxClients; i++)
	{
		if(IsClientInGame(i) && !IsFakeClient(i))
		{
			iClients++;
		}
	}

	return iClients;
}

stock int GetAllClientCount()
{
	int iClients = 0;
	for(int i = 1; i <= MaxClients; i++)
	{
		if(IsClientInGame(i))
		{
			iClients++;
		}
	}

	return iClients;
}

stock void reply(int clientId, const char[] message) {
	if (clientId > 0) {
		ReplyToCommand(clientId, message);
	} else {
		LogMessage(message);
	}
}
