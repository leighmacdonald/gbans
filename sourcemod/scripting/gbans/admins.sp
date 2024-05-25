#pragma semicolon 1
#pragma tabsize 4
#pragma newdecls required

#include "ripext"

// overrides -> groups -> admins
public void OnRebuildAdminCache(AdminCachePart part)
{
    if (!gIsAuthenticated) {
        gbLog("Tried to rebuild admin cache, but we are not authenticated yet");
        return;
    }

	if (part == AdminCache_Groups) {
        if (!gReloadGroupsQueued) {
		    RebuildGroups();
        }
	} else if (part == AdminCache_Admins) {
        if (!gReloadAdminsQueued) {
		    RebuildUsers();
        }
    } else if (part == AdminCache_Overrides) {
        if (!gReloadOverridesQueued) {
            RebuildOverrides();
        }
    }
}

void RebuildGroups() {
    char url[1024];
	makeURL("/api/sm/groups", url, sizeof url);
	
	HTTPRequest request = new HTTPRequest(url);
	addAuthHeader(request);
	
    request.Get(onRebuildGroups); 

    gReloadGroupsQueued = true;
}

void onRebuildGroups(HTTPResponse response, any value) {
    if(response.Status != HTTPStatus_OK) {
        gbLog("Invalid response code reading user groups: %d", response.Status);
		return;
    }

    JSONObject groupObj = view_as<JSONObject>(response.Data);
    JSONArray groups = view_as<JSONArray>(groupObj.Get("groups"));
    JSONArray immunities = view_as<JSONArray>(groupObj.Get("immunities"));

    int numGroups = groups.Length;

    JSONObject group;
    JSONObject groupImmunity;
    char flags[32];
	char name[128];
	int immunity;

    PrintToServer("\n\n\n\nLoading Groups\n");

    for (int i = 0; i < numGroups; i++) {
        group = view_as<JSONObject>(groups.Get(i));

        group.GetString("flags", flags, sizeof(flags));
        group.GetString("name", name, sizeof(name));
        immunity = group.GetInt("immunity_level");

        PrintToServer("Retrieved group: name: '%s' flag: '%s' immunity: '%d'", name, flags, immunity);

        GroupId grp;
		if ((grp = FindAdmGroup(name)) == INVALID_GROUP_ID)
		{
			grp = CreateAdmGroup(name);
		}
		
		/* Add flags from the database to the group */
		int num_flag_chars = strlen(flags);
		for (int j=0; j<num_flag_chars; j++)
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

    int numImmunities = immunities.Length;

    for (int i = 0; i < numImmunities; i++) {
        groupImmunity = view_as<JSONObject>(immunities.Get(i));

        char group_name[128];
        char other_name[128];
        GroupId grp, other;

        groupImmunity.GetString("group_name", group_name, sizeof(group_name));
        groupImmunity.GetString("other_name", other_name, sizeof(other_name));

        if (((grp = FindAdmGroup(group_name)) == INVALID_GROUP_ID)
			|| (other = FindAdmGroup(other_name)) == INVALID_GROUP_ID)
		{
			continue;
		}
		
		grp.AddGroupImmunity(other);

		PrintToServer("SetAdmGroupImmuneFrom(%d, %d)", grp, other);

        delete groupImmunity;

    }

    delete immunities;
    delete groups; 

    if (gReloadAdminsQueued) {
        RebuildUsers();
    }

    gReloadGroupsQueued = true;
}

void RebuildUsers() {
    char url[1024];
	makeURL("/api/sm/users", url, sizeof url);
	
	HTTPRequest request = new HTTPRequest(url);
	addAuthHeader(request);
	
    request.Get(onRebuildUsers); 

    gReloadAdminsQueued = true;
}

void onRebuildUsers(HTTPResponse response, any value) {
    if(response.Status != HTTPStatus_OK) {
        gbLog("Invalid response code reading users: %d", response.Status);
		return;
    }

    JSONObject usersObj = view_as<JSONObject>(response.Data);
    JSONArray users = view_as<JSONArray>(usersObj.Get("users"));
    JSONArray userGroups = view_as<JSONArray>(usersObj.Get("user_groups"));

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
	int id;
    
    int numUsers = users.Length;
    int numUserGroups = userGroups.Length;
    
	/* Keep track of a mapping from admin DB IDs to internal AdminIds to
	 * enable group lookups en masse */
	StringMap htAdmins = new StringMap();
	char key[16];

    PrintToServer("\n\n\nLoading Users");
    
    for (int i = 0; i < numUsers; i++) {
        user = view_as<JSONObject>(users.Get(i));

        user.GetString("authtype", authtype, sizeof(authtype));
        user.GetString("identity", identity, sizeof(identity));
        user.GetString("password", password, sizeof(password));
        user.GetString("flags", flags, sizeof(flags));
        user.GetString("name", name, sizeof(name));
        immunity = user.GetInt("immunity");
        id = user.GetInt("id");

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

		htAdmins.SetValue(key, adm);
		
		PrintToServer("User (%d,%s,%s,%s,%s,%s,%d):%d", id, authtype, identity, password, flags, name, immunity, adm);

		
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

		IntToString(userGroup.GetInt("admin_id"), key, sizeof(key));
		userGroup.GetString("group_name", group, sizeof(group));

		if (htAdmins.GetValue(key, adm))
		{
			if ((grp = FindAdmGroup(group)) == INVALID_GROUP_ID)
			{
				/* Group wasn't found, don't bother with it.  */
				continue;
			}

			adm.InheritGroup(grp);
            PrintToServer("Added user to group: %d", grp);
		}
        delete userGroup;
    }

    delete htAdmins;

    gReloadAdminsQueued = false;
}

void RebuildOverrides() {
    char url[1024];
	makeURL("/api/sm/overrides", url, sizeof url);
	
	HTTPRequest request = new HTTPRequest(url);
	addAuthHeader(request);
	
    request.Get(onRebuildOverrides);

    gReloadOverridesQueued = true;
}

void onRebuildOverrides(HTTPResponse response, any value) {
    if(response.Status != HTTPStatus_OK) {
        gbLog("Invalid response code reading overrides: %d", response.Status);
		return;
    }

    JSONArray overrides = view_as<JSONArray>(response.Data);
    JSONObject override;

    int numOverrides = overrides.Length;

    char type[64];
	char name[64];
	char flags[32];
	int flag_bits;

    PrintToServer("\n\n\nLoading OVerrides");

    for (int i = 0; i < numOverrides; i++) {
        override = view_as<JSONObject>(overrides.Get(i));

        override.GetString("type", type, sizeof(type));
        override.GetString("name", name, sizeof(name));
        override.GetString("flags", flags, sizeof(flags));

        PrintToServer("Adding override (%s, %s, %s)", type, name, flags);
        
        flag_bits = ReadFlagString(flags);
		if (StrEqual(type, "command")) {
			AddCommandOverride(name, Override_Command, flag_bits);
		} else if (StrEqual(type, "group")) {
			AddCommandOverride(name, Override_CommandGroup, flag_bits);
		}

        delete override;
    }

    if (gReloadGroupsQueued) {
        RebuildGroups();
    }

    gReloadOverridesQueued = false;
}