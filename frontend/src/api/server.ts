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

export const apiGetServerStates = async (signal: AbortSignal) =>
	await apiCall<UserServers>(signal, `/api/servers/state`);

export const apiCreateServer = async (opts: SaveServerOpts, signal: AbortSignal) =>
	transformTimeStampedDates(await apiCall<Server, SaveServerOpts>(signal, `/api/servers`, "POST", opts));

export const apiSaveServer = async (server_id: number, opts: SaveServerOpts, signal: AbortSignal) => {
	const resp = transformTimeStampedDates(
		await apiCall<Server, SaveServerOpts>(signal, `/api/servers/${server_id}`, "PUT", opts),
	);
	resp.token_created_on = parseDateTime(resp.token_created_on as unknown as string);
	return resp;
};

export const apiGetServersAdmin = async (signal: AbortSignal) => {
	const resp = await apiCall<Server[]>(signal, `/api/servers_admin`);
	return resp.map(transformTimeStampedDates).map((s) => {
		s.token_created_on = parseDateTime(s.token_created_on as unknown as string);
		return s;
	});
};

export const apiGetServers = async (signal: AbortSignal) => apiCall<ServerSimple[]>(signal, `/api/servers`);

export const apiDeleteServer = async (server_id: number, signal: AbortSignal) =>
	await apiCall(signal, `/api/servers/${server_id}`, "DELETE");

export const hasSMFlag = (flag: Flags, entity?: SMGroups | SMAdmin | SMOverrides) => {
	return entity?.flags.includes(flag) ?? false;
};

export const apiGetSMGroupImmunities = async (signal: AbortSignal) =>
	(await apiCall<SMGroupImmunity[]>(signal, "/api/smadmin/group_immunity")).map(transformCreatedOnDate);

export const apiDeleteSMGroupImmunity = async (group_immunity_id: number, signal: AbortSignal) =>
	apiCall(signal, `/api/smadmin/group_immunity/${group_immunity_id}`, "DELETE");

export const apiCreateSMGroupImmunity = async (group_id: number, other_id: number, signal: AbortSignal) =>
	transformCreatedOnDate(
		await apiCall<SMGroupImmunity>(signal, `/api/smadmin/group_immunity`, "POST", {
			group_id,
			other_id,
		}),
	);

export const apiCreateSMGroupOverrides = async (
	group_id: number,
	name: string,
	type: OverrideType,
	access: OverrideAccess,
	signal: AbortSignal,
) =>
	transformTimeStampedDates(
		await apiCall<SMGroupOverrides>(signal, `/api/smadmin/groups/${group_id}/overrides`, "POST", {
			name,
			type,
			access,
		}),
	);

export const apiSaveSMGroupOverrides = async (
	group_override_id: number,
	name: string,
	type: OverrideType,
	access: OverrideAccess,
	signal: AbortSignal,
) =>
	transformTimeStampedDates(
		await apiCall<SMGroupOverrides>(signal, `/api/smadmin/groups_overrides/${group_override_id}`, "POST", {
			name,
			type,
			access,
		}),
	);
export const apiDeleteSMGroupOverride = async (group_override_id: number, signal: AbortSignal) =>
	await apiCall(signal, `/api/smadmin/groups_overrides/${group_override_id}`, "DELETE", undefined);

export const apiGetSMOverrides = async (signal: AbortSignal) =>
	(await apiCall<SMOverrides[]>(signal, `/api/smadmin/overrides`)).map(transformTimeStampedDates);

export const apiCreateSMOverrides = async (name: string, type: OverrideType, flags: string, signal: AbortSignal) =>
	transformTimeStampedDates(
		await apiCall<SMOverrides>(signal, `/api/smadmin/overrides`, "POST", {
			name,
			type,
			flags,
		}),
	);

export const apiSaveSMOverrides = async (
	override_id: number,
	name: string,
	type: OverrideType,
	flags: string,
	signal: AbortSignal,
) =>
	transformTimeStampedDates(
		await apiCall<SMOverrides>(signal, `/api/smadmin/overrides/${override_id}`, "POST", { name, type, flags }),
	);

export const apiDeleteSMOverride = async (override_id: number, signal: AbortSignal) =>
	await apiCall(signal, `/api/smadmin/overrides/${override_id}`, "DELETE", undefined);

export const apiGetSMGroupOverrides = async (groupId: number, signal: AbortSignal) =>
	(await apiCall<SMGroupOverrides[]>(signal, `/api/smadmin/groups/${groupId}/overrides`)).map(
		transformTimeStampedDates,
	);

export const apiGetSMAdmins = async (signal: AbortSignal) =>
	(await apiCall<SMAdmin[]>(signal, "/api/smadmin/admins")).map(transformTimeStampedDates);

export const apiCreateSMAdmin = async (
	name: string,
	immunity: number,
	flags: string,
	auth_type: AuthType,
	identity: string,
	password: string,
	signal: AbortSignal,
) =>
	transformTimeStampedDates(
		await apiCall<SMAdmin>(signal, "/api/smadmin/admins", "POST", {
			name,
			immunity,
			flags,
			auth_type,
			identity,
			password,
		}),
	);

export const apiAddAdminToGroup = async (admin_id: number, group_id: number, signal: AbortSignal) =>
	transformTimeStampedDates(
		await apiCall<SMAdmin>(signal, `/api/smadmin/admins/${admin_id}/groups`, "POST", {
			group_id,
		}),
	);

export const apiDelAdminFromGroup = async (admin_id: number, group_id: number, signal: AbortSignal) =>
	transformTimeStampedDates(
		await apiCall<SMAdmin>(signal, `/api/smadmin/admins/${admin_id}/groups/${group_id}`, "DELETE"),
	);

export const apiSaveSMAdmin = async (
	admin_id: number,
	name: string,
	immunity: number,
	flags: string,
	auth_type: AuthType,
	identity: string,
	password: string,
	signal: AbortSignal,
) =>
	transformTimeStampedDates(
		await apiCall<SMAdmin>(signal, `/api/smadmin/admins/${admin_id}`, "POST", {
			name,
			immunity,
			flags,
			auth_type,
			identity,
			password,
		}),
	);

export const apiDeleteSMAdmin = async (admin_id: number, signal: AbortSignal) =>
	await apiCall(signal, `/api/smadmin/admins/${admin_id}`, "DELETE");

export const apiGetSMGroups = async (signal: AbortSignal) =>
	(await apiCall<SMGroups[]>(signal, "/api/smadmin/groups")).map(transformTimeStampedDates);

export const apiCreateSMGroup = async (name: string, immunity: number, flags: string, signal: AbortSignal) =>
	transformTimeStampedDates(
		await apiCall<SMGroups>(signal, "/api/smadmin/groups", "POST", {
			name,
			immunity,
			flags,
		}),
	);

export const apiSaveSMGroup = async (
	group_id: number,
	name: string,
	immunity: number,
	flags: string,
	signal: AbortSignal,
) =>
	transformTimeStampedDates(
		await apiCall<SMGroups>(signal, `/api/smadmin/groups/${group_id}`, "POST", {
			name,
			immunity,
			flags,
		}),
	);

export const apiDeleteSMGroup = async (group_id: number, signal: AbortSignal) =>
	await apiCall(signal, `/api/smadmin/groups/${group_id}`, "DELETE");
