#include <adminmenu>

#pragma newdecls required

public Action CmdReport(int clientId, int argc) {
	if (gReportInProgress) {
		ReplyToCommand(clientId, "A report is current in progress, please wait for it to compelte");
		return Plugin_Handled;
	}
	gReportInProgress = true;
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

public Action OnClientSayCommand(int iClient, const char[] sCommand, const char[] sArgs) {

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

public int MenuHandler_Target(Menu menu, MenuAction action, int clientId, int selectedId) {
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
			if (!GetClientAuthId(target, AuthId_SteamID64, gReportSid64, sizeof(gReportSid64), true)) {
				PrintToChat(clientId, "[SM] %t", "Player no longer available");
				return;
			}
			ShowReasonMenu(clientId);
		}
	} else if (action == MenuAction_End) {
		delete menu;
	}
}

public int MenuHandler_Reason(Menu menu, MenuAction action, int clientId, int selectedId) {
	if (action == MenuAction_Cancel || action == MenuAction_End)	{
		CloseHandle(menu);
	} else if (action == MenuAction_Select)	{
		char sInfo[64];
		GetMenuItem(menu, selectedId, sInfo, sizeof(sInfo));
		if (StrEqual(sInfo, "cheating")) {
			gReportTargetReason = cheating;
		} else if (StrEqual(sInfo, "racism")) {
			gReportTargetReason = racism;
		} else if (StrEqual(sInfo, "harassment")) {
			gReportTargetReason = harassment;
		} else if (StrEqual(sInfo, "exploiting")) {
			gReportTargetReason = exploiting;
		} else if (StrEqual(sInfo, "spam")) {
			gReportTargetReason = spam;
		} else if (StrEqual(sInfo, "languageUsed")) {
			gReportTargetReason = languageUsed;
		} else if (StrEqual(sInfo, "profile")) {
			gReportTargetReason = profile;
		} else if (StrEqual(sInfo, "itemDescriptions")) {
			gReportTargetReason = itemDescriptions;
		} else if (StrEqual(sInfo, "custom")) {
			gReportTargetReason = custom;
		} else {
			ReplyToCommand(clientId, "Unsupported reason value");
			return;
		}
		if (gReportTargetReason == custom) {
			
		} else {
			gReportInProgress = false; 
		}
	} else if (action == MenuAction_End) {
		delete menu;
	}
}
