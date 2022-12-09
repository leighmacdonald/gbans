#pragma semicolon 1
#pragma tabsize 4
#pragma newdecls required

#include <adminmenu>

const reportTimeout = 30;
const reportMinReasonLen = 10;

public
Action onCmdReport(int clientId, int argc) {
    if (gReportStartedAtTime > 0) {
        ReplyToCommand(clientId, "A report is current in progress, please wait for it to compelte");
        return Plugin_Handled;
    }
    gReportStartedAtTime = GetTime();
    ShowTargetMenu(clientId);
    return Plugin_Handled;
}

public
void ShowTargetMenu(int clientId) {
    Menu menu = CreateMenu(MenuHandler_Target);
    AddTargetsToMenu2(
        menu, 0,
        COMMAND_FILTER_CONNECTED | COMMAND_FILTER_NO_IMMUNITY | COMMAND_FILTER_NO_MULTI | COMMAND_FILTER_NO_BOTS);
    SetMenuTitle(menu, "Select A Player:");
    SetMenuExitBackButton(menu, true);
    DisplayMenu(menu, clientId, MENU_TIME_FOREVER);
}

void resetReportStatus() {
    gReportSourceId = -1;
    gReportTargetId = -1;
    gReportStartedAtTime = -1;
    gReportTargetReason = unknown;
    gReportWaitingForReason = false;
}

public
Action OnClientSayCommand(int clientId, const char[] command, const char[] args) {
    if (!gReportWaitingForReason || clientId != gReportSourceId && gReportSourceId == -1 ||
        gReportTargetReason != custom) {
        return Plugin_Continue;
    } else if (StrEqual(args, "cancel", false)) {
        PrintToChat(clientId, "Report cancelled");
        resetReportStatus();
        return Plugin_Stop;
    } else if (strlen(args) < reportMinReasonLen) {
        PrintToChat(clientId, "Report reason too short, try again or type \"cancel\" to reset");
        return Plugin_Stop;
    }
    gbLog("Got report reason: %s", args);
    report(gReportSourceId, gReportTargetId, gReportTargetReason, args);
    return Plugin_Stop;
}

public
bool report(int sourceId, int targetId, GB_BanReason reason, const char[] reasonText) {
    gbLog("Report: %d %d %d %s", sourceId, targetId, reason, reasonText);
    char sourceSid[50];
    if (!GetClientAuthId(sourceId, AuthId_Steam3, sourceSid, sizeof(sourceSid), true)) {
        ReplyToCommand(sourceId, "Failed to get sourceId of user: %d", sourceId);
        resetReportStatus();
        return false;
    }
    char targetSid[50];
    if (!GetClientAuthId(targetId, AuthId_Steam3, targetSid, sizeof(targetSid), true)) {
        ReplyToCommand(sourceId, "Failed to get targetId of user: %d", targetId);
        resetReportStatus();
        return false;
    }
    int demoTick = -1;
    char demoName[256];
    if (SourceTV_GetDemoFileName(demoName, sizeof(demoName))) {
        demoTick = SourceTV_GetRecordingTick();
    }

    JSON_Object obj = new JSON_Object();
    obj.SetString("source_id", sourceSid);
    obj.SetString("target_id", targetSid);
    obj.SetInt("reason", view_as<int>(reason));
    obj.SetString("reason_text", reasonText);
    obj.SetString("demo_name", demoName);
    obj.SetInt("demo_tick", demoTick);

    char encoded[2048];
    obj.Encode(encoded, sizeof(encoded));
    json_cleanup_and_delete(obj);

    System2HTTPRequest req = newReq(onReportRespReceived, "/api/sm/report/create");
    req.SetData(encoded);
    req.POST();
    delete req;

    resetReportStatus();

    return true;
}

void onReportRespReceived(bool success, const char[] error, System2HTTPRequest request, System2HTTPResponse response,
                          HTTPRequestMethod method) {
    if (!success) {
        gbLog("Invalid report response");
        return;
    }
    return;
}

public
void ShowReasonMenu(int clientId) {
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

public
Action Timer_checkReportState() {
    if (gReportStartedAtTime - GetTime() > reportTimeout) {
        return Plugin_Stop;
    }
    return Plugin_Continue;
}

public
int MenuHandler_Target(Menu menu, MenuAction action, int clientId, int selectedId) {
    if (action == MenuAction_Cancel || action == MenuAction_End) {
        CloseHandle(menu);
    } else if (action == MenuAction_Select) {
        int userId, targetId;
        char sTargetUserID[30];
        menu.GetItem(selectedId, sTargetUserID, sizeof(sTargetUserID));
        userId = StringToInt(sTargetUserID);

        if ((targetId = GetClientOfUserId(userId)) == 0) {
            PrintToChat(clientId, "[SM] %t", "Player no longer available");
            return -1;
        }
        gReportSourceId = clientId;
        gReportTargetId = targetId;
        ShowReasonMenu(clientId);
    } else if (action == MenuAction_End) {
        delete menu;
    }
    return 0;
}

public
int MenuHandler_Reason(Menu menu, MenuAction action, int clientId, int selectedId) {
    if (action == MenuAction_Cancel || action == MenuAction_End) {
        CloseHandle(menu);
    } else if (action == MenuAction_Select) {
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
            return -1;
        }
        if (gReportTargetReason == custom) {
            ReplyToCommand(clientId, "Enter your reason in chat, it will not be shown to others");
            gReportWaitingForReason = true;
            return 0;
        }
        return report(gReportSourceId, gReportTargetId, gReportTargetReason, "");
    } else if (action == MenuAction_End) {
        delete menu;
    }
    return 0;
}
