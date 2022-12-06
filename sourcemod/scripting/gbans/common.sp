#pragma semicolon 1

public
bool parseReason(const char[] reasonStr, GB_BanReason &reason) {
    int reasonInt = StringToInt(reasonStr, 10);
    if (reasonInt <= 0 || reasonInt < view_as<int>(custom) || reasonInt > view_as<int>(itemDescriptions)) {
        return false;
    }
    reason = view_as<GB_BanReason>(reasonInt);
    return true;
}

System2HTTPRequest newReq(System2HTTPResponseCallback cb, const char[] path) {
    char serverHost[PLATFORM_MAX_PATH];
    gHost.GetString(serverHost, sizeof(serverHost));
    int port = gPort.IntValue;
    char fullAddr[1024];
    Format(fullAddr, sizeof(fullAddr), "%s%s", serverHost, path);
    System2HTTPRequest httpRequest = new System2HTTPRequest(cb, fullAddr);
    httpRequest.SetPort(port);
    httpRequest.SetHeader("Content-Type", "application/json");
    if (strlen(gAccessToken) > 0) {
        httpRequest.SetHeader("Authorization", gAccessToken);
    }
    return httpRequest;
}

public void OnMapEnd()
{
    onMapEndStopwatch();
    onMapEndSTV();
}

stock bool isValidClient(int iClient, bool bAlive = false)
{
	if (iClient >= 1 &&
		iClient <= MaxClients &&
		IsClientConnected(iClient) &&
		IsClientInGame(iClient) &&
		!IsFakeClient(iClient) &&
		(bAlive == false || IsPlayerAlive(iClient)))
	{
		return true;
	}

	return false;
}