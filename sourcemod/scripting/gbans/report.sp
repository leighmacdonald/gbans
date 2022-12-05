#include <adminmenu>

public Action CmdReport(int clientId, int argc) {
	if (g_reportInProgress) {
		ReplyToCommand(clientId, "A report is current in progress, please wait for it to compelte");
		return Plugin_Handled;
	}
	g_reportInProgress = true;
	ShowTargetMenu(clientId);
    return Plugin_Handled;
}

public void ShowTargetMenu(int clientId) {
    Menu menu = CreateMenu(MenuHandler_Target);
	AddTargetsToMenu2(menu, 0, COMMAND_FILTER_CONNECTED|COMMAND_FILTER_NO_IMMUNITY|COMMAND_FILTER_NO_MULTI|COMMAND_FILTER_NO_BOTS);
	SetMenuTitle(menu, "Select A Player:");
	SetMenuExitBackButton(menu, true);
	DisplayMenu(menu, clientId, MENU_TIME_FOREVER);
}

public void ShowReasonMenu(int clientId) {
    Menu menu = CreateMenu(MenuHandler_Reason);
	menu.AddItem("cheating", "Cheating");
    menu.AddItem("racism", "Racism");
    menu.AddItem("harassment", "Harassment");
    menu.AddItem("exploiting", "Exploiting");
    menu.AddItem("spam", "Spam");
    menu.AddItem("languageUsed", "Language");
    menu.AddItem("profile", "Profile");
    menu.AddItem("itemDescriptions", "Items/Descriptions");
	menu.AddItem("custom", "Custom");

	SetMenuTitle(menu, "Select A Reason:");
	SetMenuExitBackButton(menu, true);
	DisplayMenu(menu, clientId, MENU_TIME_FOREVER);
}


public MenuHandler_Target(Menu menu, MenuAction action, int clientId, int selectedId) {
	if (action == MenuAction_Cancel || action == MenuAction_End) {
		CloseHandle(menu);
	} else if (action == MenuAction_Select) {
		int userid, target;
		char sTargetUserID[30];
		menu.GetItem(selectedId, sTargetUserID, sizeof(sTargetUserID));
		userid = StringToInt(sTargetUserID);

		if ((target = GetClientOfUserId(userid)) == 0) {
			PrintToChat(clientId, "[SM] %t", "Player no longer available");
			return;
		} else {
			if (!GetClientAuthId(target, AuthId_SteamID64, g_reportSid64, sizeof(g_reportSid64), true)) {
				PrintToChat(clientId, "[SM] %t", "Player no longer available");
				return;
			}
			ShowReasonMenu(clientId);
		}
	} else if (action == MenuAction_End) {
		delete menu;
	}
}

public MenuHandler_Reason(Menu menu, MenuAction action, int clientId, int selectedId) {
	if (action == MenuAction_Cancel || action == MenuAction_End)	{
		CloseHandle(menu);
	} else if (action == MenuAction_Select)	{
		char sInfo[64];
		GetMenuItem(menu, selectedId, sInfo, sizeof(sInfo));
		if (StrEqual(sInfo, "cheating")) {
			g_reportTargetReason = cheating;
		} else if (StrEqual(sInfo, "racism")) {
			g_reportTargetReason = racism;
		} else if (StrEqual(sInfo, "harassment")) {
			g_reportTargetReason = harassment;
		} else if (StrEqual(sInfo, "exploiting")) {
			g_reportTargetReason = exploiting;
		} else if (StrEqual(sInfo, "spam")) {
			g_reportTargetReason = spam;
		} else if (StrEqual(sInfo, "languageUsed")) {
			g_reportTargetReason = languageUsed;
		} else if (StrEqual(sInfo, "profile")) {
			g_reportTargetReason = profile;
		} else if (StrEqual(sInfo, "itemDescriptions")) {
			g_reportTargetReason = itemDescriptions;
		} else if (StrEqual(sInfo, "custom")) {
			g_reportTargetReason = custom;
		} else {
			ReplyToCommand(clientId, "Unsupported reason value");
			return;
		}
		if (g_reportTargetReason == custom) {
			
		} else {
			g_reportInProgress = false; 
		}
	} else if (action == MenuAction_End) {
		delete menu;
	}
}
