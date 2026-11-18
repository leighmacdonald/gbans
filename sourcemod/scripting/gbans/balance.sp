#pragma semicolon 1
#pragma tabsize 4
#pragma newdecls required

#include "globals.sp"

#define SCRAMBLE_MAX_PLAYERS 64
#define SCRAMBLE_TEAM_SPEC   1
#define SCRAMBLE_TEAM_RED    2
#define SCRAMBLE_TEAM_BLU    3

public
Action onCmdAutoTeamAction(int clientId, int argc) {
    if (GetConVarBool(gb_disable_autoteam)) {
        KickClient(clientId, "Please stop trying to stack :(");

        char auth_id[50];
        if (!GetClientAuthId(clientId, AuthId_Steam3, auth_id, sizeof auth_id, true)) {
            ReplyToCommand(clientId, "Failed to get auth_id of user: %d", clientId);
            return Plugin_Continue;
        }

        char name[64];
        if (!GetClientName(clientId, name, sizeof name)) {
            LogError("Failed to get user name?");
            return Plugin_Continue;
        }

        LogMessage("Autoteam blocked: %s [%s]", name, auth_id);
        return Plugin_Handled;
    }

    return Plugin_Continue;
}

/**
 * GB_ScrambleTeams native. Collects the real players from the supplied client
 * list, asks the backend for a balanced split, then applies the teams itself.
 */
public
any Native_GB_ScrambleTeams(Handle plugin, int numParams) {
    if (numParams < 3) {
        ThrowNativeError(SP_ERROR_PARAM, "GB_ScrambleTeams requires at least 3 arguments");
        return 0;
    }

    int requested[SCRAMBLE_MAX_PLAYERS];
    GetNativeArray(1, requested, sizeof requested);

    int numClients = GetNativeCell(2);
    if (numClients < 0 || numClients > sizeof requested) {
        ThrowNativeError(SP_ERROR_PARAM, "GB_ScrambleTeams client count must be between 0 and %d",
                         SCRAMBLE_MAX_PLAYERS);
        return 0;
    }

    Function callback = view_as<Function>(GetNativeCell(3));
    int      maxExact = numParams >= 4 ? GetNativeCell(4) : 32;

    ArrayList context = new ArrayList();
    context.Push(view_as<int>(callback));
    context.Push(0); // Real player count, filled in below.

    JSON_Array steamIds = new JSON_Array();
    int        count    = 0;

    for (int idx = 0; idx < numClients; idx++) {
        int client = requested[idx];
        // isValidClient filters out bots, SourceTV, replay clients and anyone
        // not currently in game.
        if (!isValidClient(client)) {
            continue;
        }

        char authId[32];
        if (!GetClientAuthId(client, AuthId_SteamID64, authId, sizeof authId)) {
            continue;
        }

        bool duplicate = false;
        for (int other = 0; other < count; other++) {
            char existing[32];
            context.GetString(3 + 2 * other, existing, sizeof existing);
            if (strcmp(existing, authId) == 0) {
                duplicate = true;
                break;
            }
        }
        if (duplicate) {
            continue;
        }

        // Layout: [0] callback, [1] count, then user id + steam id per player.
        context.Push(GetClientUserId(client));
        context.PushString(authId);
        steamIds.PushString(authId);
        count++;
    }
    context.Set(1, count);

    if (count == 0) {
        int noClients[SCRAMBLE_MAX_PLAYERS];
        invokeScrambleCallback(callback, false, noClients, 0, noClients, 0, noClients, 0,
                               "No valid players were supplied for the scramble");
        delete steamIds;
        delete context;
        return 0;
    }

    if (gToken[0] == '\0') {
        int noClients[SCRAMBLE_MAX_PLAYERS];
        invokeScrambleCallback(callback, false, noClients, 0, noClients, 0, noClients, 0,
                               "Server is not authenticated with gbans yet");
        authenticateServer();
        delete steamIds;
        delete context;
        return 0;
    }

    JSON_Object body = new JSON_Object();
    body.SetObject("steamIds", view_as<JSON_Object>(steamIds));
    body.SetInt("maxExact", maxExact);

    if (!postHTTPRequest("/connect/sourcemod.v1.PluginService/SMBalanceTeams", body, onBalanceTeamsResp, context)) {
        int noClients[SCRAMBLE_MAX_PLAYERS];
        invokeScrambleCallback(callback, false, noClients, 0, noClients, 0, noClients, 0,
                               "Failed to contact the gbans backend");
        delete context;
    }

    return 0;
}

void onBalanceTeamsResp(Handle request, bool failure, bool successful, EHTTPStatusCode status, any value, any unused) {
    ArrayList context  = view_as<ArrayList>(value);
    Function  callback = view_as<Function>(context.Get(0));

    int  none[SCRAMBLE_MAX_PLAYERS];
    char error[256] = "";

    if (failure || status != k_EHTTPStatusCode200OK) {
        if (status == k_EHTTPStatusCode401Unauthorized) {
            authenticateServer();
            strcopy(error, sizeof error, "Server authentication expired, please try again");
        } else {
            PrintRPCError(request, failure, status);
            strcopy(error, sizeof error, "Backend rejected the scramble request");
        }

        invokeScrambleCallback(callback, false, none, 0, none, 0, none, 0, error);
        delete context;
        delete request;
        return;
    }

    JSON_Object data = GetResponseJSON(request);
    if (data == null) {
        invokeScrambleCallback(callback, false, none, 0, none, 0, none, 0, "Invalid response from backend");
        delete context;
        delete request;
        return;
    }

    int redClients[SCRAMBLE_MAX_PLAYERS];
    int bluClients[SCRAMBLE_MAX_PLAYERS];
    int waitingClients[SCRAMBLE_MAX_PLAYERS];
    int redCount     = collectTeamClients(context, data, "redSteamIds", redClients, sizeof redClients);
    int bluCount     = collectTeamClients(context, data, "bluSteamIds", bluClients, sizeof bluClients);
    int waitingCount = collectTeamClients(context, data, "waitingSteamIds", waitingClients, sizeof waitingClients);

    for (int idx = 0; idx < redCount; idx++) {
        applyScrambleTeam(redClients[idx], SCRAMBLE_TEAM_RED);
    }
    for (int idx = 0; idx < bluCount; idx++) {
        applyScrambleTeam(bluClients[idx], SCRAMBLE_TEAM_BLU);
    }
    for (int idx = 0; idx < waitingCount; idx++) {
        applyScrambleTeam(waitingClients[idx], SCRAMBLE_TEAM_SPEC);
    }

    invokeScrambleCallback(callback, true, redClients, redCount, bluClients, bluCount, waitingClients, waitingCount,
                           error);

    json_cleanup_and_delete(data);
    delete context;
    delete request;
}

/**
 * Resolves the SteamID64s of one response team back to the client user ids
 * collected when the request was made. Disconnected players are skipped.
 */
int collectTeamClients(ArrayList context, JSON_Object data, const char[] field, int[] clients, int maxClients) {
    JSON_Array ids = view_as<JSON_Array>(data.GetObject(field));
    if (ids == null) {
        return 0;
    }

    int count = context.Get(1);
    int found = 0;
    for (int idx = 0; idx < ids.Length && found < maxClients; idx++) {
        char wanted[32];
        if (!ids.GetString(idx, wanted, sizeof wanted)) {
            continue;
        }

        for (int other = 0; other < count; other++) {
            char existing[32];
            context.GetString(3 + 2 * other, existing, sizeof existing);
            if (strcmp(existing, wanted) != 0) {
                continue;
            }

            int client = GetClientOfUserId(context.Get(2 + 2 * other));
            if (isValidClient(client)) {
                clients[found++] = client;
            }
            break;
        }
    }

    return found;
}

void applyScrambleTeam(int client, int team) {
    if (!isValidClient(client)) {
        return;
    }

    if (GetClientTeam(client) != team) {
        ChangeClientTeam(client, team);
    }

    // Force a respawn so living players actually spawn with their new team.
    if (IsPlayerAlive(client)) {
        ForcePlayerSuicide(client);
    }
}

void invokeScrambleCallback(Function callback, bool success, const int[] redClients, int redCount,
                            const int[] bluClients, int bluCount, const int[] waitingClients, int waitingCount,
                            const char[] error) {
    Call_StartFunction(INVALID_HANDLE, callback);
    Call_PushCell(success);
    Call_PushArray(redClients, SCRAMBLE_MAX_PLAYERS);
    Call_PushCell(redCount);
    Call_PushArray(bluClients, SCRAMBLE_MAX_PLAYERS);
    Call_PushCell(bluCount);
    Call_PushArray(waitingClients, SCRAMBLE_MAX_PLAYERS);
    Call_PushCell(waitingCount);
    Call_PushString(error);
    Call_Finish();
}
