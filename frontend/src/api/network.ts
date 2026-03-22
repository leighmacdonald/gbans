import type { CIDRBlockCheckResponse, CIDRBlockSource, WhitelistIP, WhitelistSteam } from "../schema/network.ts";
import { transformTimeStampedDates, transformTimeStampedDatesList } from "../util/time";
import { apiCall, type EmptyBody } from "./common";

export const apiGetNetworkUpdateDB = async (signal: AbortSignal) => apiCall(signal, "/api/network/update_db");

export const apiGetCIDRBlockListsSteamWhitelist = async (signal: AbortSignal) => {
	return transformTimeStampedDatesList(await apiCall<WhitelistSteam[]>(signal, `/api/block_list/whitelist/steam`));
};

export const apiGetCIDRBlockListsIPWhitelist = async (signal: AbortSignal) => {
	return transformTimeStampedDatesList(await apiCall<WhitelistIP[]>(signal, `/api/block_list/whitelist/ip`));
};

export const apiGetCIDRBlockLists = async (signal: AbortSignal) => {
	return transformTimeStampedDatesList(await apiCall<CIDRBlockSource[]>(signal, `/api/block_list/sources`));
};

export const apiCreateCIDRBlockSource = async (name: string, url: string, enabled: boolean, signal: AbortSignal) => {
	const resp = await apiCall<CIDRBlockSource>(signal, `/api/block_list/sources`, "POST", { name, url, enabled });
	return transformTimeStampedDates(resp);
};

export const apiUpdateCIDRBlockSource = async (
	cidr_block_source_id: number,
	name: string,
	url: string,
	enabled: boolean,
	signal: AbortSignal,
) => {
	const resp = await apiCall<CIDRBlockSource>(signal, `/api/block_list/sources/${cidr_block_source_id}`, "POST", {
		name,
		url,
		enabled,
	});
	return transformTimeStampedDates(resp);
};

export const apiDeleteCIDRBlockSource = async (cidr_block_source_id: number, signal: AbortSignal) => {
	return await apiCall<EmptyBody>(signal, `/api/block_list/sources/${cidr_block_source_id}`, "DELETE", undefined);
};

export const apiCreateWhitelistSteam = async (steam_id: string, signal: AbortSignal) => {
	const resp = await apiCall<WhitelistIP>(signal, `/api/block_list/whitelist/steam`, "POST", { steam_id });

	return transformTimeStampedDates(resp);
};

export const apiDeleteWhitelistSteam = async (steam_id: string, signal: AbortSignal) => {
	return await apiCall<EmptyBody>(signal, `/api/block_list/whitelist/steam/${steam_id}`, "DELETE", undefined);
};

export const apiCreateWhitelistIP = async (address: string, signal: AbortSignal) => {
	const resp = await apiCall<WhitelistIP>(signal, `/api/block_list/whitelist/ip`, "POST", { address });

	return transformTimeStampedDates(resp);
};

export const apiUpdateWhitelistIP = async (cidr_block_whitelist_id: number, address: string, signal: AbortSignal) => {
	const resp = await apiCall<WhitelistIP>(signal, `/api/block_list/whitelist/ip/${cidr_block_whitelist_id}`, "POST", {
		address,
	});

	return transformTimeStampedDates(resp);
};

export const apiDeleteCIDRBlockWhitelist = async (cidr_block_whitelist_id: number, signal: AbortSignal) => {
	return await apiCall<EmptyBody>(
		signal,
		`/api/block_list/whitelist/ip/${cidr_block_whitelist_id}`,
		"DELETE",
		undefined,
	);
};

export const apiCIDRBlockCheck = async (address: string, signal: AbortSignal) => {
	return await apiCall<CIDRBlockCheckResponse>(signal, `/api/block_list/checker`, "POST", { address });
};
