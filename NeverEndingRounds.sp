void:MC_ReplyToCommand(_arg0, String:_arg1[], any:_arg2)
{
	decl String:buffer[4096];
	SetGlobalTransTarget(_arg0);
	VFormat(buffer, 1024, _arg1[0], 3);
	if (_arg0 == 0)
	{
		MC_RemoveTags(buffer, 1024);
		PrintToServer("%s", buffer);
	}
	else
	{
		if (GetCmdReplySource() == 0)
		{
			MC_RemoveTags(buffer, 1024);
			PrintToConsole(_arg0, "%s", buffer);
		}
		MC_PrintToChat(_arg0, "%s", buffer);
	}
	return 0;
}

StringMap:MC_InitColorTrie()
{
	new StringMap:hTrie = StringMap.StringMap();
	StringMap.SetValue(hTrie, "aliceblue", 15792383, true);
	StringMap.SetValue(hTrie, "allies", 5077314, true);
	StringMap.SetValue(hTrie, "ancient", 15420235, true);
	StringMap.SetValue(hTrie, "antiquewhite", 16444375, true);
	StringMap.SetValue(hTrie, "aqua", 65535, true);
	StringMap.SetValue(hTrie, "aquamarine", 8388564, true);
	StringMap.SetValue(hTrie, "arcana", 11396444, true);
	StringMap.SetValue(hTrie, "axis", 16728128, true);
	StringMap.SetValue(hTrie, "azure", 32767, true);
	StringMap.SetValue(hTrie, "beige", 16119260, true);
	StringMap.SetValue(hTrie, "bisque", 16770244, true);
	StringMap.SetValue(hTrie, "black", 0, true);
	StringMap.SetValue(hTrie, "blanchedalmond", 16772045, true);
	StringMap.SetValue(hTrie, "blue", 10079487, true);
	StringMap.SetValue(hTrie, "blueviolet", 9055202, true);
	StringMap.SetValue(hTrie, "brown", 10824234, true);
	StringMap.SetValue(hTrie, "burlywood", 14596231, true);
	StringMap.SetValue(hTrie, "cadetblue", 6266528, true);
	StringMap.SetValue(hTrie, "chartreuse", 8388352, true);
	StringMap.SetValue(hTrie, "chocolate", 13789470, true);
	StringMap.SetValue(hTrie, "collectors", 11141120, true);
	StringMap.SetValue(hTrie, "common", 11584473, true);
	StringMap.SetValue(hTrie, "community", 7385162, true);
	StringMap.SetValue(hTrie, "coral", 16744272, true);
	StringMap.SetValue(hTrie, "cornflowerblue", 6591981, true);
	StringMap.SetValue(hTrie, "cornsilk", 16775388, true);
	StringMap.SetValue(hTrie, "corrupted", 10693678, true);
	StringMap.SetValue(hTrie, "crimson", 14423100, true);
	StringMap.SetValue(hTrie, "cyan", 65535, true);
	StringMap.SetValue(hTrie, "darkblue", 139, true);
	StringMap.SetValue(hTrie, "darkcyan", 35723, true);
	StringMap.SetValue(hTrie, "darkgoldenrod", 12092939, true);
	StringMap.SetValue(hTrie, "darkgray", 11119017, true);
	StringMap.SetValue(hTrie, "darkgrey", 11119017, true);
	StringMap.SetValue(hTrie, "darkgreen", 25600, true);
	StringMap.SetValue(hTrie, "darkkhaki", 12433259, true);
	StringMap.SetValue(hTrie, "darkmagenta", 9109643, true);
	StringMap.SetValue(hTrie, "darkolivegreen", 5597999, true);
	StringMap.SetValue(hTrie, "darkorange", 16747520, true);
	StringMap.SetValue(hTrie, "darkorchid", 10040012, true);
	StringMap.SetValue(hTrie, "darkred", 9109504, true);
	StringMap.SetValue(hTrie, "darksalmon", 15308410, true);
	StringMap.SetValue(hTrie, "darkseagreen", 9419919, true);
	StringMap.SetValue(hTrie, "darkslateblue", 4734347, true);
	StringMap.SetValue(hTrie, "darkslategray", 3100495, true);
	StringMap.SetValue(hTrie, "darkslategrey", 3100495, true);
	StringMap.SetValue(hTrie, "darkturquoise", 52945, true);
	StringMap.SetValue(hTrie, "darkviolet", 9699539, true);
	StringMap.SetValue(hTrie, "deeppink", 16716947, true);
	StringMap.SetValue(hTrie, "deepskyblue", 49151, true);
	StringMap.SetValue(hTrie, "dimgray", 6908265, true);
	StringMap.SetValue(hTrie, "dimgrey", 6908265, true);
	StringMap.SetValue(hTrie, "dodgerblue", 2003199, true);
	StringMap.SetValue(hTrie, "exalted", 13421773, true);
	StringMap.SetValue(hTrie, "firebrick", 11674146, true);
	StringMap.SetValue(hTrie, "floralwhite", 16775920, true);
	StringMap.SetValue(hTrie, "forestgreen", 2263842, true);
	StringMap.SetValue(hTrie, "frozen", 4817843, true);
	StringMap.SetValue(hTrie, "fuchsia", 16711935, true);
	StringMap.SetValue(hTrie, "fullblue", 255, true);
	StringMap.SetValue(hTrie, "fullred", 16711680, true);
	StringMap.SetValue(hTrie, "gainsboro", 14474460, true);
	StringMap.SetValue(hTrie, "genuine", 5076053, true);
	StringMap.SetValue(hTrie, "ghostwhite", 16316671, true);
	StringMap.SetValue(hTrie, "gold", 16766720, true);
	StringMap.SetValue(hTrie, "goldenrod", 14329120, true);
	StringMap.SetValue(hTrie, "gray", 13421772, true);
	StringMap.SetValue(hTrie, "grey", 13421772, true);
	StringMap.SetValue(hTrie, "green", 4128574, true);
	StringMap.SetValue(hTrie, "greenyellow", 11403055, true);
	StringMap.SetValue(hTrie, "haunted", 3732395, true);
	StringMap.SetValue(hTrie, "honeydew", 15794160, true);
	StringMap.SetValue(hTrie, "hotpink", 16738740, true);
	StringMap.SetValue(hTrie, "immortal", 14986803, true);
	StringMap.SetValue(hTrie, "indianred", 13458524, true);
	StringMap.SetValue(hTrie, "indigo", 4915330, true);
	StringMap.SetValue(hTrie, "ivory", 16777200, true);
	StringMap.SetValue(hTrie, "khaki", 15787660, true);
	StringMap.SetValue(hTrie, "lavender", 15132410, true);
	StringMap.SetValue(hTrie, "lavenderblush", 16773365, true);
	StringMap.SetValue(hTrie, "lawngreen", 8190976, true);
	StringMap.SetValue(hTrie, "legendary", 13839590, true);
	StringMap.SetValue(hTrie, "lemonchiffon", 16775885, true);
	StringMap.SetValue(hTrie, "lightblue", 11393254, true);
	StringMap.SetValue(hTrie, "lightcoral", 15761536, true);
	StringMap.SetValue(hTrie, "lightcyan", 14745599, true);
	StringMap.SetValue(hTrie, "lightgoldenrodyellow", 16448210, true);
	StringMap.SetValue(hTrie, "lightgray", 13882323, true);
	StringMap.SetValue(hTrie, "lightgrey", 13882323, true);
	StringMap.SetValue(hTrie, "lightgreen", 10092441, true);
	StringMap.SetValue(hTrie, "lightpink", 16758465, true);
	StringMap.SetValue(hTrie, "lightsalmon", 16752762, true);
	StringMap.SetValue(hTrie, "lightseagreen", 2142890, true);
	StringMap.SetValue(hTrie, "lightskyblue", 8900346, true);
	StringMap.SetValue(hTrie, "lightslategray", 7833753, true);
	StringMap.SetValue(hTrie, "lightslategrey", 7833753, true);
	StringMap.SetValue(hTrie, "lightsteelblue", 11584734, true);
	StringMap.SetValue(hTrie, "lightyellow", 16777184, true);
	StringMap.SetValue(hTrie, "lime", 65280, true);
	StringMap.SetValue(hTrie, "limegreen", 3329330, true);
	StringMap.SetValue(hTrie, "linen", 16445670, true);
	StringMap.SetValue(hTrie, "magenta", 16711935, true);
	StringMap.SetValue(hTrie, "maroon", 8388608, true);
	StringMap.SetValue(hTrie, "mediumaquamarine", 6737322, true);
	StringMap.SetValue(hTrie, "mediumblue", 205, true);
	StringMap.SetValue(hTrie, "mediumorchid", 12211667, true);
	StringMap.SetValue(hTrie, "mediumpurple", 9662680, true);
	StringMap.SetValue(hTrie, "mediumseagreen", 3978097, true);
	StringMap.SetValue(hTrie, "mediumslateblue", 8087790, true);
	StringMap.SetValue(hTrie, "mediumspringgreen", 64154, true);
	StringMap.SetValue(hTrie, "mediumturquoise", 4772300, true);
	StringMap.SetValue(hTrie, "mediumvioletred", 13047173, true);
	StringMap.SetValue(hTrie, "midnightblue", 1644912, true);
	StringMap.SetValue(hTrie, "mintcream", 16121850, true);
	StringMap.SetValue(hTrie, "mistyrose", 16770273, true);
	StringMap.SetValue(hTrie, "moccasin", 16770229, true);
	StringMap.SetValue(hTrie, "mythical", 8931327, true);
	StringMap.SetValue(hTrie, "navajowhite", 16768685, true);
	StringMap.SetValue(hTrie, "navy", 128, true);
	StringMap.SetValue(hTrie, "normal", 11711154, true);
	StringMap.SetValue(hTrie, "oldlace", 16643558, true);
	StringMap.SetValue(hTrie, "olive", 10404687, true);
	StringMap.SetValue(hTrie, "olivedrab", 7048739, true);
	StringMap.SetValue(hTrie, "orange", 16753920, true);
	StringMap.SetValue(hTrie, "orangered", 16729344, true);
	StringMap.SetValue(hTrie, "orchid", 14315734, true);
	StringMap.SetValue(hTrie, "palegoldenrod", 15657130, true);
	StringMap.SetValue(hTrie, "palegreen", 10025880, true);
	StringMap.SetValue(hTrie, "paleturquoise", 11529966, true);
	StringMap.SetValue(hTrie, "palevioletred", 14184595, true);
	StringMap.SetValue(hTrie, "papayawhip", 16773077, true);
	StringMap.SetValue(hTrie, "peachpuff", 16767673, true);
	StringMap.SetValue(hTrie, "peru", 13468991, true);
	StringMap.SetValue(hTrie, "pink", 16761035, true);
	StringMap.SetValue(hTrie, "plum", 14524637, true);
	StringMap.SetValue(hTrie, "powderblue", 11591910, true);
	StringMap.SetValue(hTrie, "purple", 8388736, true);
	StringMap.SetValue(hTrie, "rare", 4942335, true);
	StringMap.SetValue(hTrie, "red", 16728128, true);
	StringMap.SetValue(hTrie, "rosybrown", 12357519, true);
	StringMap.SetValue(hTrie, "royalblue", 4286945, true);
	StringMap.SetValue(hTrie, "saddlebrown", 9127187, true);
	StringMap.SetValue(hTrie, "salmon", 16416882, true);
	StringMap.SetValue(hTrie, "sandybrown", 16032864, true);
	StringMap.SetValue(hTrie, "seagreen", 3050327, true);
	StringMap.SetValue(hTrie, "seashell", 16774638, true);
	StringMap.SetValue(hTrie, "selfmade", 7385162, true);
	StringMap.SetValue(hTrie, "sienna", 10506797, true);
	StringMap.SetValue(hTrie, "silver", 12632256, true);
	StringMap.SetValue(hTrie, "skyblue", 8900331, true);
	StringMap.SetValue(hTrie, "slateblue", 6970061, true);
	StringMap.SetValue(hTrie, "slategray", 7372944, true);
	StringMap.SetValue(hTrie, "slategrey", 7372944, true);
	StringMap.SetValue(hTrie, "snow", 16775930, true);
	StringMap.SetValue(hTrie, "springgreen", 65407, true);
	StringMap.SetValue(hTrie, "steelblue", 4620980, true);
	StringMap.SetValue(hTrie, "strange", 13593138, true);
	StringMap.SetValue(hTrie, "tan", 13808780, true);
	StringMap.SetValue(hTrie, "teal", 32896, true);
	StringMap.SetValue(hTrie, "thistle", 14204888, true);
	StringMap.SetValue(hTrie, "tomato", 16737095, true);
	StringMap.SetValue(hTrie, "turquoise", 4251856, true);
	StringMap.SetValue(hTrie, "uncommon", 11584473, true);
	StringMap.SetValue(hTrie, "unique", 16766720, true);
	StringMap.SetValue(hTrie, "unusual", 8802476, true);
	StringMap.SetValue(hTrie, "valve", 10817401, true);
	StringMap.SetValue(hTrie, "vintage", 4678289, true);
	StringMap.SetValue(hTrie, "violet", 15631086, true);
	StringMap.SetValue(hTrie, "wheat", 16113331, true);
	StringMap.SetValue(hTrie, "white", 16777215, true);
	StringMap.SetValue(hTrie, "whitesmoke", 16119285, true);
	StringMap.SetValue(hTrie, "yellow", 16776960, true);
	StringMap.SetValue(hTrie, "yellowgreen", 10145074, true);
	return hTrie;
}


/* ERROR! null */
 function "C_PrintToChat" (number 2)
void:C_ReplyToCommand(_arg0, String:_arg1[], any:_arg2)
{
	decl String:szCMessage[1000];
	SetGlobalTransTarget(_arg0);
	VFormat(szCMessage, 250, _arg1[0], 3);
	if (_arg0 == 0)
	{
		C_RemoveTags(szCMessage, 250);
		PrintToServer("%s", szCMessage);
	}
	else
	{
		if (GetCmdReplySource() == 0)
		{
			C_RemoveTags(szCMessage, 250);
			PrintToConsole(_arg0, "%s", szCMessage);
		}
		C_PrintToChat(_arg0, "%s", szCMessage);
	}
	return 0;
}

void:C_RemoveTags(String:_arg0[], _arg1)
{
	new i;
	while (i < 18)
	{
		ReplaceString(_arg0[0], _arg1, C_Tag[i], "", false);
		i++;
	}
	new i;
	ReplaceString(_arg0[0], _arg1, "{teamcolor}", "", i);
	return 0;
}

bool:C_ColorAllowed(C_Colors:_arg0)
{
	if (!C_EventIsHooked)
	{
		C_SetupProfile();
		C_EventIsHooked = true;
	}
	return C_Profile_Colors[_arg0];
}

void:C_ReplaceColor(C_Colors:_arg0, C_Colors:_arg1)
{
	if (!C_EventIsHooked)
	{
		C_SetupProfile();
		C_EventIsHooked = true;
	}
	C_Profile_Colors[_arg0] = C_Profile_Colors[_arg1];
	C_Profile_TeamIndex[_arg0] = C_Profile_TeamIndex[_arg1];
	C_TagReqSayText2[_arg0] = C_TagReqSayText2[_arg1];
	Format(C_TagCode[_arg0], 2, C_TagCode[_arg1]);
	return 0;
}

C_Format(String:_arg0[], _arg1, _arg2)
{
	if (!C_EventIsHooked)
	{
		C_SetupProfile();
		HookEvent("server_spawn", 83, 2);
		C_EventIsHooked = true;
	}
	new iRandomPlayer = -1;
	if (GetEngineVersion() == 12)
	{
		Format(_arg0[0], _arg1, " %s", _arg0[0]);
	}
	if (_arg2 != -1)
	{
		if (C_Profile_SayText2)
		{
			ReplaceString(_arg0[0], _arg1, "{teamcolor}", "\x03", false);
			iRandomPlayer = _arg2;
		}
		else
		{
			ReplaceString(_arg0[0], _arg1, "{teamcolor}", C_TagCode[2], false);
		}
	}
	else
	{
		ReplaceString(_arg0[0], _arg1, "{teamcolor}", "", false);
	}
	new i;
	while (i < 18)
	{
		if (!(StrContains(_arg0[0], C_Tag[i], false) == -1))
		{
			if (C_Profile_Colors[i])
			{
				if (C_TagReqSayText2[i])
				{
					if (C_Profile_SayText2)
					{
						if (iRandomPlayer == -1)
						{
							iRandomPlayer = C_FindRandomPlayerByTeam(C_Profile_TeamIndex[i]);
							if (iRandomPlayer == -2)
							{
								ReplaceString(_arg0[0], _arg1, C_Tag[i], C_TagCode[2], false);
							}
							else
							{
								ReplaceString(_arg0[0], _arg1, C_Tag[i], C_TagCode[i], false);
							}
						}
						ThrowError("Using two team colors in one message is not allowed");
					}
					ReplaceString(_arg0[0], _arg1, C_Tag[i], C_TagCode[2], false);
				}
				ReplaceString(_arg0[0], _arg1, C_Tag[i], C_TagCode[i], false);
			}
			ReplaceString(_arg0[0], _arg1, C_Tag[i], C_TagCode[2], false);
		}
		i++;
	}
	return iRandomPlayer;
}

C_FindRandomPlayerByTeam(_arg0)
{
	if (_arg0 == 0)
	{
		return 0;
	}
	new players[MaxClients];
	new count;
	new i = 1;
	while (i <= MaxClients)
	{
		if (IsClientInGame(i))
		{
			if (_arg0 == GetClientTeam(i))
			{
				count++;
				players[0][count] = i;
			}
		}
		i++;
	}
	if (count)
	{
		return players[0][GetRandomInt(0, count - 1)];
	}
	return -2;
}


/* ERROR! null */
 function "C_SayText2" (number 9)
void:C_SetupProfile()
{
	new EngineVersion:engine = GetEngineVersion();
	if (engine == 13)
	{
		C_Profile_Colors[3] = 1;
		C_Profile_Colors[4] = 1;
		C_Profile_Colors[5] = 1;
		C_Profile_Colors[6] = 1;
		C_Profile_TeamIndex[3] = 0;
		C_Profile_TeamIndex[4] = 2;
		C_Profile_TeamIndex[5] = 3;
		C_Profile_SayText2 = true;
	}
	else
	{
		if (engine == 12)
		{
			C_Profile_Colors[4] = 1;
			C_Profile_Colors[5] = 1;
			C_Profile_Colors[6] = 1;
			C_Profile_Colors[1] = 1;
			C_Profile_Colors[7] = 1;
			C_Profile_Colors[8] = 1;
			C_Profile_Colors[9] = 1;
			C_Profile_Colors[10] = 1;
			C_Profile_Colors[11] = 1;
			C_Profile_Colors[12] = 1;
			C_Profile_Colors[13] = 1;
			C_Profile_Colors[14] = 1;
			C_Profile_Colors[15] = 1;
			C_Profile_Colors[16] = 1;
			C_Profile_Colors[17] = 1;
			C_Profile_Colors[18] = 1;
			C_Profile_TeamIndex[4] = 2;
			C_Profile_TeamIndex[5] = 3;
			C_Profile_SayText2 = true;
		}
		if (engine == 17)
		{
			C_Profile_Colors[3] = 1;
			C_Profile_Colors[4] = 1;
			C_Profile_Colors[5] = 1;
			C_Profile_Colors[6] = 1;
			C_Profile_TeamIndex[3] = 0;
			C_Profile_TeamIndex[4] = 2;
			C_Profile_TeamIndex[5] = 3;
			C_Profile_SayText2 = true;
		}
		if (!(engine == 4))
		{
			if (!(engine == 7))
			{
				if (engine == 15)
				{
					if (GetConVarBool(FindConVar("mp_teamplay")))
					{
						C_Profile_Colors[4] = 1;
						C_Profile_Colors[5] = 1;
						C_Profile_Colors[6] = 1;
						C_Profile_TeamIndex[4] = 3;
						C_Profile_TeamIndex[5] = 2;
						C_Profile_SayText2 = true;
					}
					else
					{
						C_Profile_SayText2 = false;
						C_Profile_Colors[6] = 1;
					}
				}
				if (engine == 16)
				{
					C_Profile_Colors[6] = 1;
					C_Profile_SayText2 = false;
				}
				if (GetUserMessageId("SayText2") == -1)
				{
					C_Profile_SayText2 = false;
				}
				C_Profile_Colors[4] = 1;
				C_Profile_Colors[5] = 1;
				C_Profile_TeamIndex[4] = 2;
				C_Profile_TeamIndex[5] = 3;
				C_Profile_SayText2 = true;
			}
		}
		C_Profile_Colors[3] = 1;
		C_Profile_Colors[4] = 1;
		C_Profile_Colors[5] = 1;
		C_Profile_Colors[6] = 1;
		C_Profile_TeamIndex[3] = 0;
		C_Profile_TeamIndex[4] = 3;
		C_Profile_TeamIndex[5] = 2;
		C_Profile_SayText2 = true;
	}
	return 0;
}

void:CPrintToChat(_arg0, String:_arg1[], any:_arg2)
{
	decl String:buffer[1000];
	SetGlobalTransTarget(_arg0);
	VFormat(buffer, 250, _arg1[0], 3);
	if (!g_bCFixColors)
	{
		CFixColors();
	}
	if (IsSource2009())
	{
		MC_PrintToChat(_arg0, "%s%s", g_sCPrefix, buffer);
	}
	else
	{
		C_PrintToChat(_arg0, "%s%s", g_sCPrefix, buffer);
	}
	return 0;
}


/* ERROR! null */
 function "CPrintToChatAll" (number 12)
void:CReplyToCommand(_arg0, String:_arg1[], any:_arg2)
{
	decl String:buffer[1000];
	SetGlobalTransTarget(_arg0);
	VFormat(buffer, 250, _arg1[0], 3);
	if (!g_bCFixColors)
	{
		CFixColors();
	}
	if (IsSource2009())
	{
		MC_ReplyToCommand(_arg0, "%s%s", g_sCPrefix, buffer);
	}
	else
	{
		C_ReplyToCommand(_arg0, "%s%s", g_sCPrefix, buffer);
	}
	return 0;
}

void:CFixColors()
{
	g_bCFixColors = true;
	if (!(C_ColorAllowed(3)))
	{
		if (C_ColorAllowed(7))
		{
			C_ReplaceColor(3, 7);
		}
		if (C_ColorAllowed(6))
		{
			C_ReplaceColor(3, 6);
		}
	}
	return 0;
}

bool:IsSource2009()
{
	new var1;
	return GetEngineVersion() == 13 || GetEngineVersion() == 15 || GetEngineVersion() == 16 || GetEngineVersion() == 17 || GetEngineVersion() == 19;
}

void:EmitSoundToClient(_arg0, String:_arg1[], _arg2, _arg3, _arg4, _arg5, Float:_arg6, _arg7, _arg8, Float:_arg9[3], Float:_arg10[3], bool:_arg11, Float:_arg12)
{
	new clients[1] = _arg0;
	new var1;
	if (_arg2 == -2)
	{
		var1 = _arg0;
	}
	else
	{
		var1 = _arg2;
	}
	_arg2 = var1;
	EmitSound(clients, 1, _arg1[0], _arg2, _arg3, _arg4, _arg5, _arg6, _arg7, _arg8, _arg9[0], _arg10[0], _arg11, _arg12);
	return 0;
}

bool:StrEqual(String:_arg0[], String:_arg1[], bool:_arg2)
{
	return strcmp(_arg0[0], _arg1[0], _arg2) == 0;
}

CharToLower(_arg0)
{
	if (IsCharUpper(_arg0))
	{
		return _arg0 | 32;
	}
	return _arg0;
}

StrCat(String:_arg0[], _arg1, String:_arg2[])
{
	new len = strlen(_arg0[0]);
	if (len >= _arg1)
	{
		return 0;
	}
	return Format(len + _arg0[0], _arg1 - len, "%s", _arg2[0]);
}

Protobuf:UserMessageToProtobuf(Handle:_arg0)
{
	if (GetUserMessageType() != 1)
	{
		return 0;
	}
	return _arg0;
}

BfWrite:UserMessageToBfWrite(Handle:_arg0)
{
	if (GetUserMessageType() == 1)
	{
		return 0;
	}
	return _arg0;
}

Handle:StartMessageOne(String:_arg0[], _arg1, _arg2)
{
	new players[1] = _arg1;
	return StartMessage(_arg0[0], players, 1, _arg2);
}

void:SetEntityHealth(_arg0, _arg1)
{
	static String:prop[128];
	static bool:gotconfig;
	if (!gotconfig)
	{
		new GameData:gc = 2324;
		new GameData:gc = GameData.GameData(gc);
		new bool:exists = 32;
		CloseHandle(gc);
		gc = 0;

/* ERROR! unknown load SysReq */
 function "SetEntityHealth" (number 23)
void:RespawnPlayerCallback(any:_arg0)
{
	if (_arg0)
	{
		TF2_RespawnPlayer(_arg0);
	}
	new iClient = 1;
	while (iClient <= MaxClients)
	{
		if (!g_bSoloEnabled[iClient])
		{
			if (IsValidClient(iClient))
			{
				EmitSoundToClient(iClient, "ambient/alarms/doomsday_lift_alarm.wav", -2, 0, 75, 0, ConVar.FloatValue.get(g_Cvar_HornSound), 100, -1, NULL_VECTOR, NULL_VECTOR, true, 0.0);
			}
		}
		iClient++;
	}
	return 0;
}

GetTeamAliveCount(_arg0)
{
	new iCount;
	new iClient = 1;
	while (iClient <= MaxClients)
	{
		if (IsValidClient(iClient))
		{
			if (_arg0 == GetClientTeam(iClient))
			{
				iCount++;
			}
		}
		iClient++;
	}
	return iCount;
}

GetBotClient()
{
	if (ConVar.BoolValue.get(g_Cvar_SeeBotsAsPlayers))
	{
		return 0;
	}
	new iClient = 1;
	while (iClient <= MaxClients)
	{
		if (IsClientInGame(iClient))
		{
			if (IsFakeClient(iClient))
			{
				return iClient;
			}
		}
		iClient++;
	}
	return 0;
}

GetTeamRandomAliveClient(_arg0)
{
	new iClients[MaxClients];
	new iCount;
	new iClient = 1;
	while (iClient <= MaxClients)
	{
		if (IsClientInGame(iClient))
		{
			if (_arg0 == GetClientTeam(iClient))
			{
				if (IsPlayerAlive(iClient))
				{
					iCount++;
					iClients[0][iCount] = iClient;
				}
			}
		}
		iClient++;
	}
	new var1;
	if (iCount == 0)
	{
		var1 = -1;
	}
	else
	{
		var1 = iClients[0][GetRandomInt(0, iCount - 1)];
	}
	return var1;
}

void:ToggleNER()
{
	if (g_bNERenabled)
	{
		g_bNERenabled = false;
		ConVar.SetBool(g_Cvar_NERrunning, false, false, false);
		g_fNERvoteTime = GetGameTime();
		SetConVarInt(FindConVar("mp_stalemate_enable"), 0, false, false);
		CPrintToChatAll("%t", 10572);
		return 0;
	}
	if (FindConVar("tfdb_jugg_running") != 0)
	{
		if (GetConVarBool(FindConVar("tfdb_jugg_running")))
		{
			ServerCommand("sm_jugg_disable");
		}
	}
	if (FindConVar("tfdb_pvb_running") != 0)
	{
		if (GetConVarBool(FindConVar("tfdb_pvb_running")))
		{
			ServerCommand("sm_pvb_disable");
		}
	}
	SetConVarInt(FindConVar("mp_stalemate_enable"), 1, false, false);
	g_bNERenabled = true;
	ConVar.SetBool(g_Cvar_NERrunning, true, false, false);
	g_fNERvoteTime = GetGameTime();
	CPrintToChatAll("%t", 10524);
	return 0;
}

bool:IsValidClient(_arg0)
{
	if (_arg0 > 0)
	{
		new var1;
		return IsClientInGame(_arg0) && IsPlayerAlive(_arg0);
	}
	return 0;
}

bool:IsSpectatorTeam(_arg0)
{
	return GetClientTeam(_arg0) == 1;
}

void:ChangeAliveClientTeam(_arg0, _arg1)
{
	SetEntProp(_arg0, 0, "m_lifeState", 2, 4, 0);
	ChangeClientTeam(_arg0, _arg1);
	SetEntProp(_arg0, 0, "m_lifeState", 0, 4, 0);
	new iWearable;
	new iIndex;
	while (iIndex < GetPlayerWearablesCount(_arg0)/* ERROR unknown load Call */)
	{
		iWearable = LoadEntityHandleFromAddress(iIndex * 4 + DereferencePointer(g_pMyWearables + GetEntityAddress(_arg0))/* ERROR unknown load Call */);
		if (!(iWearable == -1))
		{
			new var1;
			if (_arg1 == 3)
			{
				var1 = 1;
			}
			else
			{
				var1 = 0;
			}
			SetEntProp(iWearable, 0, "m_nSkin", var1, 4, 0);
			SetEntProp(iWearable, 0, "m_iTeamNum", _arg1, 4, 0);
		}
		iIndex++;
	}
	return 0;
}

LoadEntityHandleFromAddress(Address:_arg0)
{
	return EntRefToEntIndex(LoadFromAddress(_arg0, 2) | -2147483648);
}

Address:DereferencePointer(Address:_arg0)
{
	return LoadFromAddress(_arg0, 2);
}

GetPlayerWearablesCount(_arg0)
{
	return GetEntData(_arg0, g_pMyWearables + 12, 4);
}


/* ERROR! null */
 function "MC_PrintToChat" (number 35)

/* ERROR! null */
 function "MC_SendMessage" (number 36)
void:MC_CheckTrie()
{
	if (MC_Trie == 0)
	{
		MC_Trie = MC_InitColorTrie();
	}
	return 0;
}


/* ERROR! null */
 function "MC_ReplaceColorCodes" (number 38)
void:MC_StrToLower(String:_arg0[])
{
	new i;

/* ERROR! unknown load SysReq */
 function "MC_StrToLower" (number 39)
void:MC_RemoveTags(String:_arg0[], _arg1)
{
	MC_ReplaceColorCodes(_arg0[0], 0, true, _arg1);
	return 0;
}

public void:C_Event_MapStart(Event:_arg0, String:_arg1[], bool:_arg2)
{
	C_SetupProfile();
	new i = 1;
	while (i <= MaxClients)
	{
		C_SkipList[i] = 0;
		i++;
	}
	return 0;
}

public Action:CmdDisableNER(_arg0, _arg1)
{
	if (ConVar.BoolValue.get(g_Cvar_NERenabled))
	{
		g_bNERenabled = false;
		ConVar.SetBool(g_Cvar_NERrunning, false, false, false);
		SetConVarInt(FindConVar("mp_stalemate_enable"), 0, false, false);
		CPrintToChatAll("%t", 10220);
		return 3;
	}
	CReplyToCommand(_arg0, "%t", 10180);
	return 3;
}

public Action:CmdSolo(_arg0, _arg1)
{
	if (_arg0)
	{
		if (ConVar.BoolValue.get(g_Cvar_SoloEnabled))
		{
			if (g_bSoloEnabled[_arg0])
			{
				CPrintToChat(_arg0, "%t", 10052);
				g_bSoloEnabled[_arg0] = 0;
			}
			else
			{
				if (IsValidClient(_arg0))
				{
					if (GetTeamAliveCount(GetClientTeam(_arg0)) == 1)
					{
						CPrintToChat(_arg0, "%t", 10084);
					}
				}
				if (IsValidClient(_arg0))
				{
					if (g_bRoundStarted)
					{
						ArrayStack.Push(g_soloQueue, _arg0);
						ForcePlayerSuicide(_arg0);
					}
				}
				CPrintToChat(_arg0, "%t", 10128);
				g_bSoloEnabled[_arg0] = 1;
			}
			return 0;
		}
		CReplyToCommand(_arg0, "%t", 10028);
		return 3;
	}
	PrintToServer("Command is in game only.");
	return 3;
}

public Action:CmdToggleNER(_arg0, _arg1)
{
	if (ConVar.BoolValue.get(g_Cvar_NERenabled))
	{
		ToggleNER();
		return 3;
	}
	CReplyToCommand(_arg0, "%t", 10160);
	return 3;
}

public Action:CmdVoteNER(_arg0, _arg1)
{
	if (ConVar.BoolValue.get(g_Cvar_NERenabled))
	{
		if (g_fNERvoteTime + ConVar.FloatValue.get(g_Cvar_NERvotingTimeout) > GetGameTime())
		{
			CReplyToCommand(_arg0, "%t", "Dodgeball_NERVote_Cooldown", g_fNERvoteTime + ConVar.FloatValue.get(g_Cvar_NERvotingTimeout) - GetGameTime());
			return 3;
		}
		if (IsVoteInProgress(0))
		{
			CReplyToCommand(_arg0, "%t", 10300);
			return 3;
		}
		decl String:strMode[64];
		new var1;
		if (g_bNERenabled)
		{
			var1 = 10344;
		}
		else
		{
			var1 = 10352;
		}
		new Menu:hMenu = 28;
		new Menu:hMenu = Menu.Menu(115, hMenu);
		Menu.VoteResultCallback.set(hMenu, 117);
		Menu.SetTitle(hMenu, "%s NER mode?", strMode);
		Menu.AddItem(hMenu, "0", "Yes", 0);
		Menu.AddItem(hMenu, "1", "No", 0);
		new iTotal;
		new iClients[MaxClients];
		new iPlayer = 1;
		while (iPlayer <= MaxClients)
		{
			if (IsClientInGame(iPlayer))
			{
				if (!(IsFakeClient(iPlayer)))
				{
					iTotal++;
					iClients[0][iTotal] = iPlayer;
					iPlayer++;
				}
			}
			iPlayer++;
		}
		new iPlayer;
		Menu.DisplayVote(hMenu, iClients[0], iTotal, 10, iPlayer);
		return 3;
	}
	CReplyToCommand(_arg0, "%t", 10248);
	return 3;
}

public void:ConVarChanged(ConVar:_arg0, String:_arg1[], String:_arg2[])
{
	if (!(ConVar.BoolValue.get(g_Cvar_NERenabled)))
	{
		g_bNERenabled = false;
		CPrintToChatAll("%t", 9176);
	}
	if (!(ConVar.BoolValue.get(g_Cvar_SoloEnabled)))
	{
		new iClient = 1;
		while (iClient <= MaxClients)
		{
			if (g_bSoloEnabled[iClient])
			{
				g_bSoloEnabled[iClient] = 0;
				CPrintToChat(iClient, "%t", 9196);
			}
			iClient++;
		}
		ArrayStack.Clear(g_soloQueue);
	}
	return 0;
}

public void:OnClientDisconnect(_arg0)
{
	g_bSoloEnabled[_arg0] = 0;
	return 0;
}

public void:OnClientPutInServer(_arg0)
{
	g_bSoloEnabled[_arg0] = 0;
	if (_arg0 > 0)
	{
		if (IsClientInGame(_arg0))
		{
			SDKHook(_arg0, 2, 99);
		}
	}
	return 0;
}

public Action:OnClientTakesDamage(_arg0, &_arg1, &_arg2, &Float:_arg3, &_arg4, &_arg5, Float:_arg6[3], Float:_arg7[3])
{
	if (GetGameTime() < g_fLastRespawned + ConVar.FloatValue.get(g_Cvar_RespawnProtection))
	{
		if (IsValidClient(_arg0))
		{
			_arg3 = 0;
			return 1;
		}
	}
	return 0;
}

public void:OnConfigsExecuted()
{
	g_soloQueue = ArrayStack.ArrayStack(1);
	if (ConVar.BoolValue.get(g_Cvar_ForceNERstartMap))
	{
		g_bNERenabled = true;
	}
	HookConVarChange(g_Cvar_NERenabled, 93);
	HookConVarChange(g_Cvar_SoloEnabled, 93);
	new i = 1;
	while (i <= MaxClients)
	{
		if (IsClientInGame(i))
		{
			SDKHook(i, 2, 99);
		}
		i++;
	}
	new i = 1;
	PrecacheSound("ambient/alarms/doomsday_lift_alarm.wav", i);
	return 0;
}

public void:OnMapEnd()
{
	ArrayStack.Clear(g_soloQueue);
	CloseHandle(g_soloQueue);
	g_soloQueue = 0;
	g_fNERvoteTime = 0.0;
	return 0;
}

public void:OnMapStart()
{
	if (ConVar.BoolValue.get(g_Cvar_NERrunning))
	{
		g_bNERenabled = true;
	}
	return 0;
}

public void:OnPlayerDeath(Event:_arg0, String:_arg1[], bool:_arg2)
{
	if (g_bRoundStarted)
	{
		if (ConVar.BoolValue.get(g_Cvar_ForceNER))
		{
			g_bNERenabled = true;
		}
		new var2 = GetClientOfUserId(Event.GetInt(_arg0, "userid", 0));
		g_iLastDeadTeam = GetClientTeam(var2);
		if (g_bNERenabled)
		{
			if (GetTeamClientCount(g_iLastDeadTeam) <= 1)
			{
				if (GetTeamClientCount(g_iLastDeadTeam ^ 1) <= 1)
				{
					CPrintToChatAll("%t", 9336);
					g_fNERvoteTime = 0.0;
					g_bNERenabled = false;
				}
			}
		}
		if (g_bNERenabled)
		{
			if (GetConVarInt(FindConVar("mp_stalemate_enable")) != 1)
			{
				SetConVarInt(FindConVar("mp_stalemate_enable"), 1, false, false);
			}
		}
		if (GetTeamAliveCount(g_iLastDeadTeam) == 1)
		{
			if (g_bNERenabled)
			{
				if (GetTeamAliveCount(g_iLastDeadTeam ^ 1) > 1)
				{
					if (ConVar.BoolValue.get(g_Cvar_SoloPriority))
					{
						if (!(ArrayStack.Empty.get(g_soloQueue)))
						{
							new iSoloer;
							new iSoloer = ArrayStack.Pop(g_soloQueue, 0, iSoloer);
							while (ArrayStack.Empty.get(g_soloQueue))
							{
								if (g_bSoloEnabled[iSoloer])
								{
									if (!(IsSpectatorTeam(iSoloer)))
									{
										if (IsPlayerAlive(iSoloer))
										{
										}
										if (g_bSoloEnabled[iSoloer])
										{
											if (!(IsSpectatorTeam(iSoloer)))
											{
												if (!(IsPlayerAlive(iSoloer)))
												{
													ChangeClientTeam(iSoloer, g_iLastDeadTeam);
													TF2_RespawnPlayer(iSoloer);
													g_fLastRespawned = GetGameTime();
													EmitSoundToClient(iSoloer, "ambient/alarms/doomsday_lift_alarm.wav", -2, 0, 75, 0, ConVar.FloatValue.get(g_Cvar_HornSound), 100, -1, NULL_VECTOR, NULL_VECTOR, true, 0.0);
													return 0;
												}
											}
										}
									}
								}
								iSoloer = ArrayStack.Pop(g_soloQueue, 0, false);
							}
							if (g_bSoloEnabled[iSoloer])
							{
								if (!(IsSpectatorTeam(iSoloer)))
								{
									if (!(IsPlayerAlive(iSoloer)))
									{
										ChangeClientTeam(iSoloer, g_iLastDeadTeam);
										TF2_RespawnPlayer(iSoloer);
										g_fLastRespawned = GetGameTime();
										EmitSoundToClient(iSoloer, "ambient/alarms/doomsday_lift_alarm.wav", -2, 0, 75, 0, ConVar.FloatValue.get(g_Cvar_HornSound), 100, -1, NULL_VECTOR, NULL_VECTOR, true, 0.0);
										return 0;
									}
								}
							}
						}
					}
					new iRandomOpponent = GetTeamRandomAliveClient(g_iLastDeadTeam ^ 1);
					g_iOldTeam[iRandomOpponent] = g_iLastDeadTeam ^ 1;
					if (IsValidClient(iRandomOpponent))
					{
						ChangeAliveClientTeam(iRandomOpponent, g_iLastDeadTeam);
					}
					return 0;
				}
			}
			if (!(ArrayStack.Empty.get(g_soloQueue)))
			{
				new iSoloer;
				new iSoloer = ArrayStack.Pop(g_soloQueue, 0, iSoloer);
				while (ArrayStack.Empty.get(g_soloQueue))
				{
					if (g_bSoloEnabled[iSoloer])
					{
						if (!(IsSpectatorTeam(iSoloer)))
						{
							if (IsPlayerAlive(iSoloer))
							{
							}
							if (g_bSoloEnabled[iSoloer])
							{
								if (!(IsSpectatorTeam(iSoloer)))
								{
									if (!(IsPlayerAlive(iSoloer)))
									{
										ChangeClientTeam(iSoloer, g_iLastDeadTeam);
										TF2_RespawnPlayer(iSoloer);
										g_fLastRespawned = GetGameTime();
										EmitSoundToClient(iSoloer, "ambient/alarms/doomsday_lift_alarm.wav", -2, 0, 75, 0, ConVar.FloatValue.get(g_Cvar_HornSound), 100, -1, NULL_VECTOR, NULL_VECTOR, true, 0.0);
										return 0;
									}
								}
							}
						}
					}
					iSoloer = ArrayStack.Pop(g_soloQueue, 0, false);
				}
				if (g_bSoloEnabled[iSoloer])
				{
					if (!(IsSpectatorTeam(iSoloer)))
					{
						if (!(IsPlayerAlive(iSoloer)))
						{
							ChangeClientTeam(iSoloer, g_iLastDeadTeam);
							TF2_RespawnPlayer(iSoloer);
							g_fLastRespawned = GetGameTime();
							EmitSoundToClient(iSoloer, "ambient/alarms/doomsday_lift_alarm.wav", -2, 0, 75, 0, ConVar.FloatValue.get(g_Cvar_HornSound), 100, -1, NULL_VECTOR, NULL_VECTOR, true, 0.0);
							return 0;
						}
					}
				}
			}
			new var3 = 0;
			GetMapTimeLeft(var3);
			if (g_bNERenabled)
			{
				if (var3 + 10 <= 10)
				{
					new iEnt = -1;
					new iEnt = CreateEntityByName("game_round_win", iEnt);
					SetVariantInt(g_iLastDeadTeam ^ 1);
					AcceptEntityInput(iEnt, "SetTeam", -1, -1, 0);
					AcceptEntityInput(iEnt, "RoundWin", -1, -1, 0);
					return 0;
				}
			}
			if (g_bNERenabled)
			{
				if (GetTeamAliveCount(g_iLastDeadTeam ^ 1) == 1)
				{
					if (GetBotClient() != g_iBot)
					{
						return 0;
					}
					decl String:buffer[2048];
					decl String:namebuffer[256];
					new iTotalPlayers;
					ArrayStack.Clear(g_soloQueue);
					new iWinner = GetTeamRandomAliveClient(g_iLastDeadTeam ^ 1);
					new iMarkedSoloer;
					new iPlayer = 1;
					while (iPlayer <= MaxClients)
					{
						if (IsClientInGame(iPlayer))
						{
							if (!(IsSpectatorTeam(iPlayer)))
							{
								new iLifeState;

/* ERROR! unknown load SysReq */
 function "OnPlayerDeath" (number 53)
public void:OnPluginStart()
{
	LoadTranslations("ner.txt");
	RegAdminCmd("sm_ner", 89, 2, "Forcefully toggle NER (Never ending rounds)", "", 0);
	RegAdminCmd("sm_ner_disable", 85, 2, "Forcefully disable NER (Never ending rounds)", "", 0);
	RegConsoleCmd("sm_votener", 91, "Vote to toggle NER", 0);
	RegConsoleCmd("sm_solo", 87, "Toggle solo mode", 0);
	g_fNERvoteTime = 0.0;
	g_Cvar_NERvotingTimeout = CreateConVar("tfdb_NERvotingTimeout", "120", "Voting timeout for NER", 0, true, 0.0, false, 0.0);
	g_Cvar_ForceNER = CreateConVar("tfdb_ForceNER", "0", "Forces NER mode (when possible), can't be disabled anymore", 0, true, 0.0, true, 1.0);
	g_Cvar_ForceNERstartMap = CreateConVar("tfdb_ForceNERstartMap", "0", "Enables NER mode at the start of the map", 0, true, 0.0, true, 1.0);
	g_Cvar_NERenabled = CreateConVar("tfdb_NERenabled", "1", "Enables/disables NER", 0, true, 0.0, true, 1.0);
	g_Cvar_SoloEnabled = CreateConVar("tfdb_SoloEnabled", "1", "Enables/disables Solo", 0, true, 0.0, true, 1.0);
	g_Cvar_SoloPriority = CreateConVar("tfdb_SoloPriority", "1", "Gives solo players priority before NER players", 0, true, 0.0, true, 1.0);
	g_Cvar_HornSound = CreateConVar("tfdb_HornSoundLevel", "0.5", "Volume level of the horn played when respawning players", 0, true, 0.0, true, 1.0);
	g_Cvar_RespawnProtection = CreateConVar("tfdb_RespawnProtection", "3.0", "Amount of time that a player is protected for after respawning", 0, true, 0.0, false, 0.0);
	g_Cvar_NERrunning = CreateConVar("tfdb_NERrunning", "0", "This is not a changable Cvar!", 0, true, 0.0, true, 1.0);
	g_Cvar_SeeBotsAsPlayers = CreateConVar("NER_BotDebug", "0", "Makes it so that NER plugin ignores bots & view them as players.", 0, true, 0.0, true, 1.0);
	g_pMyWearables = FindSendPropInfo("CTFPlayer", "m_hMyWearables", 0, 0, 0, 0);
	HookEvent("arena_round_start", 113, 2);
	HookEvent("player_death", 107, 0);
	HookEvent("teamplay_round_win", 111, 2);
	HookEvent("teamplay_round_stalemate", 111, 2);
	return 0;
}

public void:OnRoundEnd(Event:_arg0, String:_arg1[], bool:_arg2)
{
	g_bRoundStarted = false;
	return 0;
}

public void:OnSetupFinished(Event:_arg0, String:_arg1[], bool:_arg2)
{
	ArrayStack.Clear(g_soloQueue);
	g_fLastRespawned = 0.0;
	g_iBot = GetBotClient();
	decl String:buffer[2048];
	decl String:nameBuffer[256];
	new iRedTeamCount = 2;
	new iRedTeamCount = GetTeamClientCount(iRedTeamCount);
	new iBlueTeamCount = 3;
	new iBlueTeamCount = GetTeamClientCount(iBlueTeamCount);
	new iClient = 1;
	while (iClient <= MaxClients)
	{
		if (IsValidClient(iClient))
		{
			if (g_bSoloEnabled[iClient])
			{
				if (!(IsSpectatorTeam(iClient)))
				{
					new var1;
					if (GetClientTeam(iClient) == 2)
					{
						iRedTeamCount--;
						var1 = iRedTeamCount;
					}
					else
					{
						iBlueTeamCount--;
						var1 = iBlueTeamCount;
					}
					if (var1 > 0)
					{
						if (ArrayStack.Empty.get(g_soloQueue))
						{
							Format(nameBuffer, 64, "%N", iClient);
						}
						else
						{
							Format(nameBuffer, 64, ", %N", iClient);
						}
						StrCat(buffer, 512, nameBuffer);
						ArrayStack.Push(g_soloQueue, iClient);
						ForcePlayerSuicide(iClient);
					}
					g_bSoloEnabled[iClient] = 0;
					CPrintToChat(iClient, "%t", 9240);
				}
			}
		}
		iClient++;
	}
	if (!(ArrayStack.Empty.get(g_soloQueue)))
	{
		CPrintToChatAll("%t", "Dodgeball_Solo_Announce_All_Soloers", buffer);
	}
	g_bRoundStarted = true;
	g_fNERvoteTime = 0.0;
	return 0;
}

public VoteMenuHandler(Menu:_arg0, MenuAction:_arg1, _arg2, _arg3)
{
	if (_arg1 == 16)
	{
		CloseHandle(_arg0);
		_arg0 = 0;
	}
	return 0;
}

public void:VoteResultHandler(Menu:_arg0, _arg1, _arg2, _arg3[][], _arg4, _arg5[][])
{
	new iWinnerIndex;
	if (_arg4 > 1)
	{
		if (_arg5[0][1] + 4/* ERROR unknown load Binary */ == _arg5[0][0] + 4/* ERROR unknown load Binary */)
		{
			iWinnerIndex = GetRandomInt(0, 1);
		}
	}
	decl String:strWinner[32];

/* ERROR! Can't print expression: Heap */
 function "VoteResultHandler" (number 58)
public void:__ext_core_SetNTVOptional()
{
	MarkNativeAsOptional("GetFeatureStatus");
	MarkNativeAsOptional("RequireFeature");
	MarkNativeAsOptional("AddCommandListener");
	MarkNativeAsOptional("RemoveCommandListener");
	MarkNativeAsOptional("BfWriteBool");
	MarkNativeAsOptional("BfWriteByte");
	MarkNativeAsOptional("BfWriteChar");
	MarkNativeAsOptional("BfWriteShort");
	MarkNativeAsOptional("BfWriteWord");
	MarkNativeAsOptional("BfWriteNum");
	MarkNativeAsOptional("BfWriteFloat");
	MarkNativeAsOptional("BfWriteString");
	MarkNativeAsOptional("BfWriteEntity");
	MarkNativeAsOptional("BfWriteAngle");
	MarkNativeAsOptional("BfWriteCoord");
	MarkNativeAsOptional("BfWriteVecCoord");
	MarkNativeAsOptional("BfWriteVecNormal");
	MarkNativeAsOptional("BfWriteAngles");
	MarkNativeAsOptional("BfReadBool");
	MarkNativeAsOptional("BfReadByte");
	MarkNativeAsOptional("BfReadChar");
	MarkNativeAsOptional("BfReadShort");
	MarkNativeAsOptional("BfReadWord");
	MarkNativeAsOptional("BfReadNum");
	MarkNativeAsOptional("BfReadFloat");
	MarkNativeAsOptional("BfReadString");
	MarkNativeAsOptional("BfReadEntity");
	MarkNativeAsOptional("BfReadAngle");
	MarkNativeAsOptional("BfReadCoord");
	MarkNativeAsOptional("BfReadVecCoord");
	MarkNativeAsOptional("BfReadVecNormal");
	MarkNativeAsOptional("BfReadAngles");
	MarkNativeAsOptional("BfGetNumBytesLeft");
	MarkNativeAsOptional("BfWrite.WriteBool");
	MarkNativeAsOptional("BfWrite.WriteByte");
	MarkNativeAsOptional("BfWrite.WriteChar");
	MarkNativeAsOptional("BfWrite.WriteShort");
	MarkNativeAsOptional("BfWrite.WriteWord");
	MarkNativeAsOptional("BfWrite.WriteNum");
	MarkNativeAsOptional("BfWrite.WriteFloat");
	MarkNativeAsOptional("BfWrite.WriteString");
	MarkNativeAsOptional("BfWrite.WriteEntity");
	MarkNativeAsOptional("BfWrite.WriteAngle");
	MarkNativeAsOptional("BfWrite.WriteCoord");
	MarkNativeAsOptional("BfWrite.WriteVecCoord");
	MarkNativeAsOptional("BfWrite.WriteVecNormal");
	MarkNativeAsOptional("BfWrite.WriteAngles");
	MarkNativeAsOptional("BfRead.ReadBool");
	MarkNativeAsOptional("BfRead.ReadByte");
	MarkNativeAsOptional("BfRead.ReadChar");
	MarkNativeAsOptional("BfRead.ReadShort");
	MarkNativeAsOptional("BfRead.ReadWord");
	MarkNativeAsOptional("BfRead.ReadNum");
	MarkNativeAsOptional("BfRead.ReadFloat");
	MarkNativeAsOptional("BfRead.ReadString");
	MarkNativeAsOptional("BfRead.ReadEntity");
	MarkNativeAsOptional("BfRead.ReadAngle");
	MarkNativeAsOptional("BfRead.ReadCoord");
	MarkNativeAsOptional("BfRead.ReadVecCoord");
	MarkNativeAsOptional("BfRead.ReadVecNormal");
	MarkNativeAsOptional("BfRead.ReadAngles");
	MarkNativeAsOptional("BfRead.BytesLeft.get");
	MarkNativeAsOptional("PbReadInt");
	MarkNativeAsOptional("PbReadFloat");
	MarkNativeAsOptional("PbReadBool");
	MarkNativeAsOptional("PbReadString");
	MarkNativeAsOptional("PbReadColor");
	MarkNativeAsOptional("PbReadAngle");
	MarkNativeAsOptional("PbReadVector");
	MarkNativeAsOptional("PbReadVector2D");
	MarkNativeAsOptional("PbGetRepeatedFieldCount");
	MarkNativeAsOptional("PbSetInt");
	MarkNativeAsOptional("PbSetFloat");
	MarkNativeAsOptional("PbSetBool");
	MarkNativeAsOptional("PbSetString");
	MarkNativeAsOptional("PbSetColor");
	MarkNativeAsOptional("PbSetAngle");
	MarkNativeAsOptional("PbSetVector");
	MarkNativeAsOptional("PbSetVector2D");
	MarkNativeAsOptional("PbAddInt");
	MarkNativeAsOptional("PbAddFloat");
	MarkNativeAsOptional("PbAddBool");
	MarkNativeAsOptional("PbAddString");
	MarkNativeAsOptional("PbAddColor");
	MarkNativeAsOptional("PbAddAngle");
	MarkNativeAsOptional("PbAddVector");
	MarkNativeAsOptional("PbAddVector2D");
	MarkNativeAsOptional("PbRemoveRepeatedFieldValue");
	MarkNativeAsOptional("PbReadMessage");
	MarkNativeAsOptional("PbReadRepeatedMessage");
	MarkNativeAsOptional("PbAddMessage");
	MarkNativeAsOptional("Protobuf.ReadInt");
	MarkNativeAsOptional("Protobuf.ReadInt64");
	MarkNativeAsOptional("Protobuf.ReadFloat");
	MarkNativeAsOptional("Protobuf.ReadBool");
	MarkNativeAsOptional("Protobuf.ReadString");
	MarkNativeAsOptional("Protobuf.ReadColor");
	MarkNativeAsOptional("Protobuf.ReadAngle");
	MarkNativeAsOptional("Protobuf.ReadVector");
	MarkNativeAsOptional("Protobuf.ReadVector2D");
	MarkNativeAsOptional("Protobuf.GetRepeatedFieldCount");
	MarkNativeAsOptional("Protobuf.SetInt");
	MarkNativeAsOptional("Protobuf.SetInt64");
	MarkNativeAsOptional("Protobuf.SetFloat");
	MarkNativeAsOptional("Protobuf.SetBool");
	MarkNativeAsOptional("Protobuf.SetString");
	MarkNativeAsOptional("Protobuf.SetColor");
	MarkNativeAsOptional("Protobuf.SetAngle");
	MarkNativeAsOptional("Protobuf.SetVector");
	MarkNativeAsOptional("Protobuf.SetVector2D");
	MarkNativeAsOptional("Protobuf.AddInt");
	MarkNativeAsOptional("Protobuf.AddInt64");
	MarkNativeAsOptional("Protobuf.AddFloat");
	MarkNativeAsOptional("Protobuf.AddBool");
	MarkNativeAsOptional("Protobuf.AddString");
	MarkNativeAsOptional("Protobuf.AddColor");
	MarkNativeAsOptional("Protobuf.AddAngle");
	MarkNativeAsOptional("Protobuf.AddVector");
	MarkNativeAsOptional("Protobuf.AddVector2D");
	MarkNativeAsOptional("Protobuf.RemoveRepeatedFieldValue");
	MarkNativeAsOptional("Protobuf.ReadMessage");
	MarkNativeAsOptional("Protobuf.ReadRepeatedMessage");
	MarkNativeAsOptional("Protobuf.AddMessage");
	VerifyCoreVersion();
	return 0;
}

