#pragma semicolon 1
#pragma tabsize 4
#pragma newdecls required

#include "common.sp"

public Action onAdminCmdReload(int clientId, int argc)
{
	reloadAdmins(true);

	return Plugin_Handled;
}

public void reloadAdmins(bool force)
{
	ServerCommand("sm_reloadadmins");
}

public void OnClientPostAdminCheck(int clientId) {
	gbLog("--- OnClientPostAdminCheck");
	if (!(clientId > 0 && IsClientInGame(clientId) && !IsFakeClient(clientId))) {
		return;
	}

	checkPlayer(clientId);
}

public void authenticateServer() {
    if (gAuthWaiting) {
        gbLog("Waiting to reauthenticate gbans");
        return;
    }

    gAuthWaiting = true;
	gbLog("Authenticating gbans...");

	char passwd[40];
	gbCoreServerKey.GetString(passwd, sizeof passwd);

	JSONObject req = new JSONObject();
	req.SetString("password", passwd);

	postHTTPRequest("/connect/sourcemod.v1.PluginService/SMAuthenticate", req, onAuthenticate);
}

void onAuthenticate(HTTPResponse response, any value) {
	switch(response.Status) {
		case HTTPStatus_OK: {
			JSONObject data = view_as<JSONObject>(response.Data);
			data.GetString("token", gToken, sizeof gToken);
			gbLog("Authenticated server successfully");

			reloadAdmins(true);
		}
		default: {
			gbLog("Got invalid auth response: %d", response.Status);
		}
	}

	gAuthWaiting = false;
}

void checkPlayer(int clientId)
{
	if (!(clientId > 0 && IsClientInGame(clientId) && !IsFakeClient(clientId))) {
		gbLog("Skipping check on invalid player");
		return ;
	}

	char ip[16];
	GetClientIP(clientId, ip, sizeof ip);

	char name[32];
	GetClientName(clientId, name, sizeof name);

	char clientAuth[64];
	GetClientAuthId(clientId, AuthId_SteamID64, clientAuth, sizeof(clientAuth));

	JSONObject obj = new JSONObject();
	obj.SetString("steamId", clientAuth);
	obj.SetInt("clientId", clientId);
	obj.SetString("ip", ip);
	obj.SetString("name", name);

	postHTTPRequest("/connect/sourcemod.v1.PluginService/SMCheck", obj, onCheckResp);
}

void onCheckResp(HTTPResponse response, any value) {
	gbLog("--- onCheckResp");
	switch (response.Status) {
	case HTTPStatus_OK:
		// good boi
		return;
	case HTTPStatus_Forbidden: {
		JSONObject data = view_as<JSONObject>(response.Data);

		char msg[256];
		data.GetString("msg", msg, sizeof msg);
		int clientId = data.GetInt("clientId");
		int banType = data.GetInt("banType");

		switch(banType) {
			case BSNoComm: {
				if(!BaseComm_IsClientMuted(clientId)) {
					BaseComm_SetClientMute(clientId, true);
				}
				if(!BaseComm_IsClientGagged(clientId)){
					BaseComm_SetClientGag(clientId, true);
				}
				ReplyToCommand(clientId, "You are currently muted/gag, it will expire automatically");
				gbLog("Muted \"%L\" for an unfinished mute punishment.", clientId);
			}
			case BSNetwork: {
				KickClient(clientId, msg);
				LogAction(0, clientId, "Kicked \"%L\" for a network block.", clientId);
			}
			case BSBanned: {
				KickClient(clientId, msg);
				LogAction(0, clientId, "Kicked \"%L\" for an unfinished ban.", clientId);
			}
		}
	}
	default: {
		LogError("Invalid check response code: %d", response.Status);
	}
	}
}
