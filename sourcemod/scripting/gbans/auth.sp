#pragma semicolon 1
#pragma tabsize 4
#pragma newdecls required

#include "globals.sp"

/**
Authenicates the server with the backend API system.

Send unauthenticated request for token to -> API /api/server_auth
Recv Token <- API
Send authenticated commands with header "Authorization $token" set for subsequent calls -> API /api/<path>

*/
void refreshToken() {
    char serverName[PLATFORM_MAX_PATH];
    gServerName.GetString(serverName, sizeof(serverName));

    char serverKey[PLATFORM_MAX_PATH];
    gServerKey.GetString(serverKey, sizeof(serverKey));

    JSON_Object obj = new JSON_Object();
    obj.SetString("server_name", serverName);
    obj.SetString("key", serverKey);
    char encoded[1024];
    obj.Encode(encoded, sizeof(encoded));
    json_cleanup_and_delete(obj);

    System2HTTPRequest req = newReq(OnAuthReqReceived, "/api/server/auth");
    req.SetData(encoded);
    req.POST();
    delete req;
}

void OnAuthReqReceived(bool success, const char[] error, System2HTTPRequest request, System2HTTPResponse response,
                       HTTPRequestMethod method) {
    if (success) {
        char lastURL[128];
        response.GetLastURL(lastURL, sizeof(lastURL));
        int statusCode = response.StatusCode;
        if (statusCode != HTTP_STATUS_OK) {
            PrintToServer("[GB] Bad status on authentication request: %d", statusCode);
            return;
        }
        char[] content = new char[response.ContentLength + 1];
        response.GetContent(content, response.ContentLength + 1);
        JSON_Object resp = json_decode(content);
        bool ok = resp.GetBool("status");
        if (!ok) {
            PrintToServer("[GB] Invalid response status, cannot authenticate");
            return;
        }
        JSON_Object data = resp.GetObject("result");
        char token[512];
        data.GetString("token", token, sizeof(token));
        if (strlen(token) == 0) {
            PrintToServer("[GB] Invalid response status, invalid token");
            return;
        }
        gAccessToken = token;
        PrintToServer("[GB] Successfully authenticated with gbans server");
        json_cleanup_and_delete(resp);
        reloadAdmins();
    } else {
        PrintToServer("[GB] Error on authentication request: %s", error);
    }
}

public
Action AdminCmdReload(int clientId, int argc) {
    reloadAdmins();
    return Plugin_Handled;
}

void reloadAdmins() {
    PrintToServer("[GB] Refreshing admin users");
    System2HTTPRequest req = newReq(OnAdminsReqReceived, "/api/server/admins");
    req.GET();
    delete req;
}

void OnAdminsReqReceived(bool success, const char[] error, System2HTTPRequest request, System2HTTPResponse response,
                       HTTPRequestMethod method) {
    if (success) {
        char lastURL[128];
        response.GetLastURL(lastURL, sizeof(lastURL));
        int statusCode = response.StatusCode;
        if (statusCode != HTTP_STATUS_OK) {
            PrintToServer("[GB] Bad status on reload admins request: %d", statusCode);
            return;
        }
        char[] content = new char[response.ContentLength + 1];
        response.GetContent(content, response.ContentLength + 1);
        JSON_Object resp = json_decode(content);
        bool ok = resp.GetBool("status");
        if (!ok) {
            PrintToServer("[GB] Invalid response status, cannot reload admins");
            return;
        }
        JSON_Array adminArray = view_as<JSON_Array>(resp.GetObject("result"));

        int length = adminArray.Length;
        AdminId adm;
        int immunity;
        for (int i = 0; i < length; i += 1) {
            char flags[32];
            char steamId[32];
            JSON_Object perm = adminArray.GetObject(i);
            perm.GetString("flags", flags, sizeof(flags));
            perm.GetString("steam_id", steamId, sizeof(steamId));

            if ((adm = FindAdminByIdentity(AUTHMETHOD_STEAM, steamId)) == INVALID_ADMIN_ID)
            {
                // "" = anon admin
                adm = CreateAdmin("");
                if (!adm.BindIdentity(AUTHMETHOD_STEAM, steamId))
                {
                    LogError("Could not bind prefetched SQL admin (identity \"%s\")", steamId);
                    continue;
                }
            }

            /* Apply each flag */
            int len = strlen(flags);
            AdminFlag flag;
            for (int j=0; j<len; j++)
            {
                if (!FindFlagByChar(flags[j], flag))
                {
                    continue;
                }
                adm.SetFlag(flag, true);
            }
            adm.ImmunityLevel = immunity;
        }       

        PrintToServer("[GB] Successfully reloaded %d admins", length);
        json_cleanup_and_delete(resp);
    } else {
        PrintToServer("[GB] Error on reload admins request: %s", error);
    }
}

public
void OnClientPostAdminCheck(int clientId) {
    switch (gPlayers[clientId].banType) {
        case BSNoComm: {
            if (!BaseComm_IsClientMuted(clientId)) {
                BaseComm_SetClientMute(clientId, true);
            }
            if (!BaseComm_IsClientGagged(clientId)) {
                BaseComm_SetClientGag(clientId, true);
            }
            ReplyToCommand(clientId, "You are currently muted/gag, it will expire automatically");
            LogAction(0, clientId, "Muted \"%L\" for an unfinished mute punishment.", clientId);
        }
        case BSBanned: {
            KickClient(clientId, gPlayers[clientId].message);
            LogAction(0, clientId, "Kicked \"%L\" for an unfinished ban.", clientId);
        }
    }
}


void CheckPlayer(int clientId, const char[] auth, const char[] ip, const char[] name) {
    if (!IsClientConnected(clientId) || IsFakeClient(clientId)) {
        PrintToServer("[GB] Skipping check on invalid player");
        return;
    }
    char encoded[1024];
    JSON_Object obj = new JSON_Object();
    obj.SetString("steam_id", auth);
    obj.SetInt("client_id", clientId);
    obj.SetString("ip", ip);
    obj.SetString("name", name);
    obj.Encode(encoded, sizeof(encoded));
    json_cleanup_and_delete(obj);

    System2HTTPRequest req = newReq(OnCheckResp, "/api/check");
    req.SetData(encoded);
    req.POST();
    delete req;
}

void OnCheckResp(bool success, const char[] error, System2HTTPRequest request, System2HTTPResponse response,
                 HTTPRequestMethod method) {
    if (success) {
        char lastURL[128];
        response.GetLastURL(lastURL, sizeof(lastURL));
        int statusCode = response.StatusCode;
        char[] content = new char[response.ContentLength + 1];
        response.GetContent(content, response.ContentLength + 1);
        if (statusCode != HTTP_STATUS_OK) {
            // Fail open if the server is broken
            return;
        }
        
        JSON_Object resp = json_decode(content);
        JSON_Object data = resp.GetObject("result");
        int clientId = data.GetInt("client_id");
        int banType = data.GetInt("ban_type");
        int permissionLevel = data.GetInt("permission_level");
        char msg[256]; // welcome or ban message
        data.GetString("msg", msg, sizeof(msg));
        if(IsFakeClient(clientId)) {
            return;
        }
        char ip[16];
        GetClientIP(clientId, ip, sizeof(ip));
        gPlayers[clientId].authed = true;
        gPlayers[clientId].ip = ip;
        gPlayers[clientId].banType = banType;
        gPlayers[clientId].message = msg;
        gPlayers[clientId].permissionLevel = permissionLevel;

        PrintToServer("[GB] Client authenticated (banType: %d level: %d)", banType, permissionLevel);      
        json_cleanup_and_delete(resp);  
    } else {
        PrintToServer("[GB] Error on authentication request: %s", error);
    }
}
