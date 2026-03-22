import {
	type AppealQueryFilter,
	AppealState,
	type AppealStateEnum,
	type BanOpts,
	type BanRecord,
	BanType,
	type BanTypeEnum,
	type BodyMDMessage,
	type sbBanRecord,
	type UnbanPayload,
	type UpdateBanPayload,
} from "../schema/bans.ts";
import type { TimeStampedWithValidUntil } from "../schema/chrono.ts";
import type { BanQueryOpts } from "../schema/query.ts";
import type { BanAppealMessage } from "../schema/report.ts";
import {
	parseDateTime,
	transformCreatedOnDate,
	transformTimeStampedDates,
	transformTimeStampedDatesList,
} from "../util/time.ts";
import { apiCall } from "./common";

export const appealStateString = (as: AppealStateEnum): string => {
	switch (as) {
		case AppealState.Any:
			return "Any";
		case AppealState.Open:
			return "Open";
		case AppealState.Denied:
			return "Denied";
		case AppealState.Accepted:
			return "Accepted";
		case AppealState.Reduced:
			return "Reduced";
		default:
			return "No Appeal";
	}
};

export const banTypeString = (bt: BanTypeEnum) => {
	switch (bt) {
		case BanType.Banned:
			return "Banned";
		case BanType.NoComm:
			return "Muted";
		default:
			return "Not Banned";
	}
};

export const apiGetBans = async (opts: BanQueryOpts, signal: AbortSignal) => {
	const resp = await apiCall<BanRecord[], BanQueryOpts>(signal, `/api/bans`, "GET", opts);
	return resp.map(transformTimeStampedDates);
};

export const apiGetBansSteamBySteamID = async (steam_id: string, signal: AbortSignal) => {
	const resp = await apiCall<BanRecord[], BanQueryOpts>(signal, `/api/bans/all/${steam_id}`, "GET");
	return resp.map(transformTimeStampedDates);
};

export const apiGetBanBySteam = async (steamID: string, signal: AbortSignal) =>
	transformTimeStampedDates(await apiCall<BanRecord>(signal, `/api/bans/steamid/${steamID}`));

export function applyDateTime<T>(row: T & TimeStampedWithValidUntil) {
	const record = {
		...row,
		created_on: parseDateTime(row.created_on as unknown as string),
		updated_on: parseDateTime(row.updated_on as unknown as string),
	};
	if (record?.valid_until) {
		record.valid_until = parseDateTime(record.valid_until as unknown as string);
	}
	return record;
}

export const apiGetBanSteam = async (ban_id: number, deleted = false, signal: AbortSignal) => {
	const resp = await apiCall<BanRecord>(signal, `/api/ban/${ban_id}?deleted=${deleted}`);

	return resp ? transformTimeStampedDates(resp) : undefined;
};

export const apiGetAppeals = async (opts: AppealQueryFilter, signal: AbortSignal) => {
	const appeals = await apiCall<BanRecord[], AppealQueryFilter>(signal, `/api/appeals`, "POST", opts);
	return appeals.map(applyDateTime);
};

export const apiCreateBan = async (p: BanOpts, signal: AbortSignal) =>
	transformTimeStampedDates(await apiCall<BanRecord, BanOpts>(signal, `/api/bans`, "POST", p));

export const apiUpdateBanSteam = async (ban_id: number, payload: UpdateBanPayload, signal: AbortSignal) =>
	transformTimeStampedDates(
		await apiCall<BanRecord, UpdateBanPayload>(signal, `/api/ban/${ban_id}`, "POST", payload),
	);

export const apiDeleteBan = async (ban_id: number, unban_reason_text: string, signal: AbortSignal) =>
	await apiCall<null, UnbanPayload>(signal, `/api/ban/${ban_id}`, "DELETE", {
		unban_reason_text,
	});

export const apiGetBanMessages = async (ban_id: number, signal: AbortSignal) => {
	const resp = await apiCall<BanAppealMessage[]>(signal, `/api/bans/${ban_id}/messages`);

	return transformTimeStampedDatesList(resp);
};

export const apiCreateBanMessage = async (ban_id: number, body_md: string, signal: AbortSignal) => {
	const resp = await apiCall<BanAppealMessage, BodyMDMessage>(signal, `/api/bans/${ban_id}/messages`, "POST", {
		body_md,
	});

	return transformTimeStampedDates(resp);
};

export const apiUpdateBanMessage = async (ban_message_id: number, message: string, signal: AbortSignal) =>
	transformTimeStampedDates(
		await apiCall<BanAppealMessage>(signal, `/api/bans/message/${ban_message_id}`, "POST", {
			body_md: message,
		}),
	);

export const apiDeleteBanMessage = async (ban_message_id: number, signal: AbortSignal) =>
	await apiCall(signal, `/api/bans/message/${ban_message_id}`, "DELETE", {});

export const apiSetBanAppealState = async (ban_id: number, appeal_state: AppealStateEnum, signal: AbortSignal) =>
	await apiCall(signal, `/api/ban/${ban_id}/status`, "POST", {
		appeal_state,
	});

export const apiGetSourceBans = async (steam_id: string, signal: AbortSignal) => {
	const resp = await apiCall<sbBanRecord[]>(signal, `/api/sourcebans/${steam_id}`, "GET");
	return resp.map(transformCreatedOnDate);
};
