import type {
	ConnectionQuery,
	DiscordUser,
	IPQuery,
	MessageQuery,
	NetworkDetails,
	PermissionUpdate,
	Person,
	PersonConnection,
	PersonMessage,
	PersonSettings,
	PlayerProfile,
	PlayerQuery,
	SteamValidate,
	UserNotification,
	UserProfile,
} from "../schema/people.ts";
import type { LazyResult } from "../util/table.ts";
import { parseDateTime, transformCreatedOnDate, transformTimeStampedDates } from "../util/time.ts";
import { apiCall } from "./common";

export const apiGetSteamValidate = async (query: string, signal: AbortSignal) => {
	return await apiCall<SteamValidate>(signal, `/api/steam/validate?query=${query}`);
};

export const apiGetProfile = async (query: string, signal: AbortSignal) => {
	const profile = await apiCall<PlayerProfile>(signal, `/api/profile?query=${query}`);
	profile.player = transformTimeStampedDates(profile.player);
	return profile;
};

export const apiGetCurrentProfile = async (signal: AbortSignal) => apiCall<UserProfile>(signal, `/api/current_profile`);

export const apiLogout = async (signal: AbortSignal) => await apiCall(signal, `/api/auth/logout`);

export const apiSearchPeople = async (opts: PlayerQuery, signal: AbortSignal) => {
	const people = await apiCall<LazyResult<Person>>(signal, `/api/players`, "GET", opts);
	people.data = people.data.map(transformTimeStampedDates);
	return people;
};

export const apiGetMessageContext = async (messageId: number, padding: number = 5, signal: AbortSignal) => {
	const resp = await apiCall<PersonMessage[]>(signal, `/api/message/${messageId}/context/${padding}`);
	return resp.map((msg) => {
		return {
			...msg,
			created_on: parseDateTime(msg.created_on as unknown as string),
		};
	});
};

export const apiGetMessages = async (opts: MessageQuery, signal: AbortSignal) => {
	const resp = await apiCall<LazyResult<PersonMessage>, MessageQuery>(signal, `/api/messages`, "GET", {
		...opts,
		// date_start: (opts.date_start ?? "") === "" ? undefined : opts.date_start,
		// date_end: (opts.date_end ?? "") === "" ? undefined : opts.date_end,
	});

	resp.data = resp.data.map((msg) => {
		return {
			...msg,
			created_on: parseDateTime(msg.created_on as unknown as string),
		};
	});

	return resp;
};

export const apiGetNotifications = async (signal: AbortSignal) => {
	const resp = await apiCall<UserNotification[]>(signal, `/api/notifications`);
	return resp.map((msg) => {
		return {
			...msg,
			created_on: parseDateTime(msg.created_on as unknown as string),
		};
	});
};

export const apiNotificationsMarkAllRead = async (signal: AbortSignal) => {
	return await apiCall(signal, `/api/notifications/all`, "POST");
};

export const apiNotificationsMarkRead = async (message_ids: number[], signal: AbortSignal) => {
	return await apiCall(signal, `/api/notifications`, "POST", { message_ids });
};

export const apiNotificationsDeleteAll = async (signal: AbortSignal) => {
	return await apiCall(signal, `/api/notifications/all`, "DELETE");
};

export const apiNotificationsDelete = async (message_ids: number[], signal: AbortSignal) => {
	return await apiCall(signal, `/api/notifications`, "DELETE", { message_ids });
};

export const apiGetConnections = async (opts: ConnectionQuery, signal: AbortSignal) => {
	const resp = await apiCall<LazyResult<PersonConnection>, ConnectionQuery>(signal, `/api/connections`, "GET", opts);

	resp.data = resp.data.map(transformCreatedOnDate);
	return resp;
};

export const apiGetNetworkDetails = async (opts: IPQuery, signal: AbortSignal) => {
	return await apiCall<NetworkDetails, IPQuery>(signal, `/api/network`, "POST", opts);
};

export const apiUpdatePlayerPermission = async (steamId: string, args: PermissionUpdate, signal: AbortSignal) =>
	transformTimeStampedDates(
		await apiCall<Person, PermissionUpdate>(signal, `/api/player/${steamId}/permissions`, "PUT", args),
	);

export const apiGetPersonSettings = async (signal: AbortSignal) => {
	return transformTimeStampedDates(await apiCall<PersonSettings>(signal, `/api/current_profile/settings`));
};

export const apiSavePersonSettings = async (
	forum_signature: string,
	forum_profile_messages: boolean,
	stats_hidden: boolean,
	center_projectiles: boolean,
	signal: AbortSignal,
) => {
	return transformTimeStampedDates(
		await apiCall<PersonSettings>(signal, `/api/current_profile/settings`, "POST", {
			forum_signature,
			forum_profile_messages,
			stats_hidden,
			center_projectiles,
		}),
	);
};

export const apiDiscordUser = async (signal: AbortSignal) => {
	return transformTimeStampedDates(await apiCall<DiscordUser>(signal, "/api/discord/user"));
};

export const discordAvatarURL = (user: DiscordUser) => {
	return `https://cdn.discordapp.com/avatars/${user.id}/${user.avatar}.png`;
};

export const apiDiscordLogout = async (signal: AbortSignal) => {
	return await apiCall<DiscordUser>(signal, "/api/discord/logout");
};
