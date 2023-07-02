public void onPluginStart()
{
	CreateTimer(30.0, updateState, _, TIMER_REPEAT);
}


stock GetRealClientCount()
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


stock GetAllClientCount()
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


public Action updateState(Handle timer)
{
    ConVar sv_visiblemaxplayers = FindConVar("sv_visiblemaxplayers");
	char curmap[256];
	if(!GetCurrentMap(curmap, sizeof curmap))
	{
		return Plugin_Handled;
	}

	char encoded[1024];
	JSON_Object obj = new JSON_Object();
	obj.SetString("current_map", curmap);
	obj.SetInt("players_real", GetRealClientCount());
    obj.SetInt("players_total", GetAllClientCount());
    obj.SetInt("players_visible", sv_visiblemaxplayers.IntValue);
	obj.Encode(encoded, sizeof encoded);
	json_cleanup_and_delete(obj);

	System2HTTPRequest req = newReq(onStateUpdateResp, "/api/state_update");
	req.SetData(encoded);
	req.POST();
	delete req;
    
    return Plugin_Handled;
}

void onStateUpdateResp(bool success, const char[] error, System2HTTPRequest request, System2HTTPResponse response, HTTPRequestMethod method)
{
	if(!success)
	{
		gbLog("State update successfully");

	}
	else
	{
		gbLog("State update error: %s", error);
	}
}
