/**
 * Implements a HTTP version of the standard sourcemod  admin-sql-prefetch plugin.
 */
#pragma semicolon 1
#pragma tabsize 4
#pragma newdecls required

#include "ripext"

// Are we already running admin update
bool gQueuedAdminUpdate = false;

// Naively expects to be called in the order of: overrides -> groups -> admins
// TODO improve call logic
public void OnRebuildAdminCache(AdminCachePart part)
{
    if (gQueuedAdminUpdate) {
        return;
    }

    gQueuedAdminUpdate = true;
    if (gToken[0] == '\n') {
        authenticateServer();
    } else {
        RebuildGroups();
    }
}

void RebuildGroups() {
    postHTTPRequest("/connect/sourcemod.v1.PluginService/SMGroups", new JSONObject(), onRebuildGroups);
}

void onRebuildGroups(HTTPResponse response, any value) {
    printJSON(response.Data);
    if (response.Status != HTTPStatus_OK) {
        gbLog("Invalid response code reading user groups: %d", response.Status);
		return;
    }

    JSONObject groupObj = view_as<JSONObject>(response.Data);
    if (!groupObj.HasKey("groups")) {
        return;
    }

    JSONArray groups = view_as<JSONArray>(groupObj.Get("groups"));

    int numGroups = groups.Length;

    JSONObject group;
    JSONObject groupImmunity;
    char flags[32];
	char name[128];
	int immunity;

    for (int i = 0; i < numGroups; i++) {
        group = view_as<JSONObject>(groups.Get(i));

        group.GetString("flags", flags, sizeof(flags));
        group.GetString("name", name, sizeof(name));
        immunity = group.GetInt("immunityLevel");


        GroupId grp;
		if ((grp = FindAdmGroup(name)) == INVALID_GROUP_ID)
		{
			grp = CreateAdmGroup(name);
		}

		/* Add flags from the database to the group */
		int numFlagChars = strlen(flags);
		for (int j=0; j<numFlagChars; j++)
		{
			AdminFlag flag;
			if (!FindFlagByChar(flags[j], flag))
			{
				continue;
			}
			grp.SetFlag(flag, true);
		}

		/* Set the immunity level this group has */
		grp.ImmunityLevel = immunity;

        delete group;
    }

    if (groupObj.HasKey("immunities")) {
        JSONArray immunities = view_as<JSONArray>(groupObj.Get("immunities"));

        int numImmunities = immunities.Length;

        for (int i = 0; i < numImmunities; i++) {
            groupImmunity = view_as<JSONObject>(immunities.Get(i));

            char groupName[128];
            char otherName[128];
            GroupId grp, other;

            groupImmunity.GetString("groupName", groupName, sizeof(groupName));
            groupImmunity.GetString("otherName", otherName, sizeof(otherName));

            if (((grp = FindAdmGroup(groupName)) == INVALID_GROUP_ID)
    			|| (other = FindAdmGroup(otherName)) == INVALID_GROUP_ID)
    		{
    			continue;
    		}

    		grp.AddGroupImmunity(other);

    #if defined _DEBUG
    		PrintToServer("SetAdmGroupImmuneFrom(%d, %d)", grp, other);
    #endif

            delete groupImmunity;
        }

        delete immunities;
    }

    delete groups;

    gbLog("Loaded %d groups", numGroups);

    RebuildUsers();
}

void RebuildUsers() {
     postHTTPRequest("/connect/sourcemod.v1.PluginService/SMUsers", new JSONObject(), onRebuildUsers);
}

void onRebuildUsers(HTTPResponse response, any value) {
    printJSON(response.Data);
    if(response.Status != HTTPStatus_OK) {
        gbLog("Invalid response code reading users: %d", response.Status);
        gQueuedAdminUpdate = false;
		return;
    }

    JSONObject usersObj = view_as<JSONObject>(response.Data);
    JSONArray users = view_as<JSONArray>(usersObj.Get("users"));
    JSONArray userGroups = view_as<JSONArray>(usersObj.Get("userGroups"));

    JSONObject user;
    JSONObject userGroup;
    char authtype[16];
	char identity[80];
	char password[80];
	char flags[32];
	char name[80];
	int immunity;
	AdminId adm;
	GroupId grp;


    int numUsers = users.Length;
    int numUserGroups = userGroups.Length;

	/* Keep track of a mapping from admin DB IDs to internal AdminIds to
	 * enable group lookups en masse */
	StringMap htAdmins = new StringMap();
	char key[16];

    for (int i = 0; i < numUsers; i++) {
        user = view_as<JSONObject>(users.Get(i));

        user.GetString("authType", authtype, sizeof(authtype));
        user.GetString("identity", identity, sizeof(identity));
        user.GetString("password", password, sizeof(password));
        user.GetString("flags", flags, sizeof(flags));
        user.GetString("name", name, sizeof(name));
        if (user.HasKey("immunity")) {
            immunity = user.GetInt("immunity");
        } else {
            immunity = 0;
        }

        /* Use a pre-existing admin if we can */
		if ((adm = FindAdminByIdentity(authtype, identity)) == INVALID_ADMIN_ID)
		{
			adm = CreateAdmin(name);
			if (!adm.BindIdentity(authtype, identity))
			{
				LogError("Could not bind prefetched SQL admin (authtype \"%s\") (identity \"%s\")", authtype, identity);
				continue;
			}
		}

        IntToString(user.GetInt("id"), key, sizeof(key));

        htAdmins.SetValue(key, adm);

		/* See if this admin wants a password */
		if (password[0] != '\0')
		{
			adm.SetPassword(password);
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

        delete user;
    }

    char group[80];
    for (int i = 0; i < numUserGroups; i++) {
        userGroup = view_as<JSONObject>(userGroups.Get(i));

		IntToString(userGroup.GetInt("adminId"), key, sizeof(key));
		userGroup.GetString("groupName", group, sizeof(group));


		if (htAdmins.GetValue(key, adm))
		{
			if ((grp = FindAdmGroup(group)) == INVALID_GROUP_ID)
			{
				/* Group wasn't found, don't bother with it.  */
                gbLog("Failed to add group, it doesnt exist: %s", group);
				continue;
			}

			adm.InheritGroup(grp);
		}

        delete userGroup;
    }

    delete htAdmins;

    gbLog("Loaded %d users into %d groups", numUsers, numUserGroups);

    RebuildOverrides();
}

void RebuildOverrides() {
    postHTTPRequest("/connect/sourcemod.v1.PluginService/SMOverrides", new JSONObject(), onRebuildOverrides);

}

void onRebuildOverrides(HTTPResponse response, any value) {
    printJSON(response.Data);
    if(response.Status != HTTPStatus_OK) {
        gbLog("Invalid response code reading overrides: %d", response.Status);
        gQueuedAdminUpdate = false;
		return;
    }

    JSONArray overrides = view_as<JSONArray>(response.Data);
    JSONObject override;

    int numOverrides = overrides.Length;

    char type[64];
	char name[64];
	char flags[32];
	int flagBits;

    for (int i = 0; i < numOverrides; i++) {
        override = view_as<JSONObject>(overrides.Get(i));

        override.GetString("type", type, sizeof(type));
        override.GetString("name", name, sizeof(name));
        override.GetString("flags", flags, sizeof(flags));

#if defined _DEBUG
        PrintToServer("Adding override (%s, %s, %s)", type, name, flags);
#endif

        flagBits = ReadFlagString(flags);
		if (StrEqual(type, "command")) {
			AddCommandOverride(name, Override_Command, flagBits);
		} else if (StrEqual(type, "group")) {
			AddCommandOverride(name, Override_CommandGroup, flagBits);
		}

        delete override;
    }

    gbLog("Loaded %d overrides", numOverrides);

    gQueuedAdminUpdate = false;

}
