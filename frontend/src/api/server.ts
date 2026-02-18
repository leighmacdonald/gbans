import type { SaveServerOpts, Server, ServerSimple, UserServers } from "../schema/server.ts";
import type {
	AuthType,
	Flags,
	OverrideAccess,
	OverrideType,
	SMAdmin,
	SMGroupImmunity,
	SMGroupOverrides,
	SMGroups,
	SMOverrides,
} from "../schema/sourcemod.ts";
import { parseDateTime, transformCreatedOnDate, transformTimeStampedDates } from "../util/time.ts";
import { apiCall } from "./common";

export const cleanMapName = (name: string): string => {
	if (!name.startsWith("workshop/")) {
		return name;
	}
	const a = name.split("/");
	if (a.length !== 2) {
		return name;
	}
	const b = a[1].split(".ugc");
	if (a.length !== 2) {
		return name;
	}
	return b[0];
};

export const apiGetServerStates = async (abortController?: AbortController) =>
	await apiCall<UserServers>(`/api/servers/state`, "GET", undefined, abortController);

export const apiCreateServer = async (opts: SaveServerOpts) =>
	transformTimeStampedDates(await apiCall<Server, SaveServerOpts>(`/api/servers`, "POST", opts));

export const apiSaveServer = async (server_id: number, opts: SaveServerOpts) => {
	const resp = transformTimeStampedDates(
		await apiCall<Server, SaveServerOpts>(`/api/servers/${server_id}`, "PUT", opts),
	);
	resp.token_created_on = parseDateTime(resp.token_created_on as unknown as string);
	return resp;
};

export const apiGetServersAdmin = async (abortController?: AbortController) => {
	const resp = await apiCall<Server[]>(`/api/servers_admin`, "GET", undefined, abortController);
	return resp.map(transformTimeStampedDates).map((s) => {
		s.token_created_on = parseDateTime(s.token_created_on as unknown as string);
		return s;
	});
};

export const apiGetServers = async () => apiCall<ServerSimple[]>(`/api/servers`, "GET", undefined);

export const apiDeleteServer = async (server_id: number) => await apiCall(`/api/servers/${server_id}`, "DELETE");

export const hasSMFlag = (flag: Flags, entity?: SMGroups | SMAdmin | SMOverrides) => {
	return entity?.flags.includes(flag) ?? false;
};

export const apiGetSMGroupImmunities = async () =>
	(await apiCall<SMGroupImmunity[]>("/api/smadmin/group_immunity")).map(transformCreatedOnDate);

export const apiDeleteSMGroupImmunity = async (group_immunity_id: number) =>
	apiCall(`/api/smadmin/group_immunity/${group_immunity_id}`, "DELETE");

export const apiCreateSMGroupImmunity = async (group_id: number, other_id: number) =>
	transformCreatedOnDate(
		await apiCall<SMGroupImmunity>(`/api/smadmin/group_immunity`, "POST", {
			group_id,
			other_id,
		}),
	);

export const apiCreateSMGroupOverrides = async (
	group_id: number,
	name: string,
	type: OverrideType,
	access: OverrideAccess,
) =>
	transformTimeStampedDates(
		await apiCall<SMGroupOverrides>(`/api/smadmin/groups/${group_id}/overrides`, "POST", { name, type, access }),
	);

export const apiSaveSMGroupOverrides = async (
	group_override_id: number,
	name: string,
	type: OverrideType,
	access: OverrideAccess,
) =>
	transformTimeStampedDates(
		await apiCall<SMGroupOverrides>(`/api/smadmin/groups_overrides/${group_override_id}`, "POST", {
			name,
			type,
			access,
		}),
	);
export const apiDeleteSMGroupOverride = async (group_override_id: number) =>
	await apiCall(`/api/smadmin/groups_overrides/${group_override_id}`, "DELETE", undefined);

export const apiGetSMOverrides = async () =>
	(await apiCall<SMOverrides[]>(`/api/smadmin/overrides`, "GET")).map(transformTimeStampedDates);

export const apiCreateSMOverrides = async (name: string, type: OverrideType, flags: string) =>
	transformTimeStampedDates(
		await apiCall<SMOverrides>(`/api/smadmin/overrides`, "POST", {
			name,
			type,
			flags,
		}),
	);

export const apiSaveSMOverrides = async (override_id: number, name: string, type: OverrideType, flags: string) =>
	transformTimeStampedDates(
		await apiCall<SMOverrides>(`/api/smadmin/overrides/${override_id}`, "POST", { name, type, flags }),
	);

export const apiDeleteSMOverride = async (override_id: number) =>
	await apiCall(`/api/smadmin/overrides/${override_id}`, "DELETE", undefined);

export const apiGetSMGroupOverrides = async (groupId: number) =>
	(await apiCall<SMGroupOverrides[]>(`/api/smadmin/groups/${groupId}/overrides`, "GET")).map(
		transformTimeStampedDates,
	);

export const apiGetSMAdmins = async () =>
	(await apiCall<SMAdmin[]>("/api/smadmin/admins")).map(transformTimeStampedDates);

export const apiCreateSMAdmin = async (
	name: string,
	immunity: number,
	flags: string,
	auth_type: AuthType,
	identity: string,
	password: string,
) =>
	transformTimeStampedDates(
		await apiCall<SMAdmin>("/api/smadmin/admins", "POST", {
			name,
			immunity,
			flags,
			auth_type,
			identity,
			password,
		}),
	);

export const apiAddAdminToGroup = async (admin_id: number, group_id: number) =>
	transformTimeStampedDates(
		await apiCall<SMAdmin>(`/api/smadmin/admins/${admin_id}/groups`, "POST", {
			group_id,
		}),
	);

export const apiDelAdminFromGroup = async (admin_id: number, group_id: number) =>
	transformTimeStampedDates(
		await apiCall<SMAdmin>(`/api/smadmin/admins/${admin_id}/groups/${group_id}`, "DELETE", undefined),
	);

export const apiSaveSMAdmin = async (
	admin_id: number,
	name: string,
	immunity: number,
	flags: string,
	auth_type: AuthType,
	identity: string,
	password: string,
) =>
	transformTimeStampedDates(
		await apiCall<SMAdmin>(`/api/smadmin/admins/${admin_id}`, "POST", {
			name,
			immunity,
			flags,
			auth_type,
			identity,
			password,
		}),
	);

export const apiDeleteSMAdmin = async (admin_id: number) =>
	await apiCall(`/api/smadmin/admins/${admin_id}`, "DELETE", undefined);

export const apiGetSMGroups = async () =>
	(await apiCall<SMGroups[]>("/api/smadmin/groups")).map(transformTimeStampedDates);

export const apiCreateSMGroup = async (name: string, immunity: number, flags: string) =>
	transformTimeStampedDates(
		await apiCall<SMGroups>("/api/smadmin/groups", "POST", {
			name,
			immunity,
			flags,
		}),
	);

export const apiSaveSMGroup = async (group_id: number, name: string, immunity: number, flags: string) =>
	transformTimeStampedDates(
		await apiCall<SMGroups>(`/api/smadmin/groups/${group_id}`, "POST", {
			name,
			immunity,
			flags,
		}),
	);

export const apiDeleteSMGroup = async (group_id: number) =>
	await apiCall(`/api/smadmin/groups/${group_id}`, "DELETE", undefined);
