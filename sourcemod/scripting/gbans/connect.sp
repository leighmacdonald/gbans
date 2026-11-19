#pragma semicolon 1
#pragma tabsize 4
#pragma newdecls required

#define NET_CHAN_LIMIT_DISABLED 0

/**
 * Disable the channel rate limit before hibernation so the first player can
 * complete the slow cold-start handshake without being dropped. A negative
 * gb_net_chan_limit_active disables this behaviour entirely.
 */
public
void OnServerEnterHibernation() {
    if (GetConVarInt(gb_net_chan_limit_active) < 0) {
        return;
    }

    ConVar limit = FindConVar("net_chan_limit_msec");
    if (limit == null) {
        return;
    }

    SetConVarInt(limit, NET_CHAN_LIMIT_DISABLED);
    LogMessage("Server entering hibernation, net_chan_limit_msec disabled for cold start");
}

/**
 * Once a real player is fully in the server the cold start is over, so put the
 * normal channel rate limit back in place.
 */
public
void OnClientPutInServer(int client) {
    int activeLimit = GetConVarInt(gb_net_chan_limit_active);
    if (activeLimit < 0 || !isValidClient(client)) {
        return;
    }

    ConVar limit = FindConVar("net_chan_limit_msec");
    if (limit == null || GetConVarInt(limit) == activeLimit) {
        return;
    }

    SetConVarInt(limit, activeLimit);
    LogMessage("Restored net_chan_limit_msec to %d after client %d connected", activeLimit, client);
}

public
Action Event_PlayerConnect(Event event, const char[] name, bool dontBroadcast) {
    event.BroadcastDisabled = GetConVarBool(gb_hide_connections);
    return Plugin_Continue;
}

public
Action Event_PlayerDisconnect(Event event, const char[] name, bool dontBroadcast) {
    event.BroadcastDisabled = GetConVarBool(gb_hide_connections);
    return Plugin_Continue;
}
