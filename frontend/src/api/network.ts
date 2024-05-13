import { apiCall, EmptyBody, TimeStamped, transformTimeStampedDates, transformTimeStampedDatesList } from './common';

export interface CIDRBlockSource extends TimeStamped {
    cidr_block_source_id: number;
    name: string;
    url: string;
    enabled: boolean;
}

export const apiGetCIDRBlockListsSteamWhitelist = async (abortController?: AbortController) => {
    return transformTimeStampedDatesList(
        await apiCall<WhitelistSteam[]>(`/api/block_list/whitelist/steam`, 'GET', undefined, abortController)
    );
};

export const apiGetCIDRBlockListsIPWhitelist = async (abortController?: AbortController) => {
    return transformTimeStampedDatesList(
        await apiCall<WhitelistIP[]>(`/api/block_list/whitelist/ip`, 'GET', undefined, abortController)
    );
};

export const apiGetCIDRBlockLists = async (abortController?: AbortController) => {
    return transformTimeStampedDatesList(
        await apiCall<CIDRBlockSource[]>(`/api/block_list/sources`, 'GET', undefined, abortController)
    );
};

export const apiCreateCIDRBlockSource = async (
    name: string,
    url: string,
    enabled: boolean,
    abortController?: AbortController
) => {
    const resp = await apiCall<CIDRBlockSource>(
        `/api/block_list/sources`,
        'POST',
        { name, url, enabled },
        abortController
    );
    return transformTimeStampedDates(resp);
};

export const apiUpdateCIDRBlockSource = async (
    cidr_block_source_id: number,
    name: string,
    url: string,
    enabled: boolean,
    abortController?: AbortController
) => {
    const resp = await apiCall<CIDRBlockSource>(
        `/api/block_list/sources/${cidr_block_source_id}`,
        'POST',
        { name, url, enabled },
        abortController
    );
    return transformTimeStampedDates(resp);
};

export const apiDeleteCIDRBlockSource = async (cidr_block_source_id: number, abortController?: AbortController) => {
    return await apiCall<EmptyBody>(
        `/api/block_list/sources/${cidr_block_source_id}`,
        'DELETE',
        undefined,
        abortController
    );
};

export interface WhitelistIP extends TimeStamped {
    cidr_block_whitelist_id: number;
    address: string;
}

export interface WhitelistSteam extends TimeStamped {
    steam_id: string;
    personaname: string;
    avatar_hash: string;
}

export const apiCreateWhitelistSteam = async (steam_id: string, abortController?: AbortController) => {
    const resp = await apiCall<WhitelistIP>(`/api/block_list/whitelist/steam`, 'POST', { steam_id }, abortController);

    return transformTimeStampedDates(resp);
};

export const apiDeleteWhitelistSteam = async (steam_id: string, abortController?: AbortController) => {
    return await apiCall<EmptyBody>(
        `/api/block_list/whitelist/steam/${steam_id}`,
        'DELETE',
        undefined,
        abortController
    );
};

export const apiCreateWhitelistIP = async (address: string, abortController?: AbortController) => {
    const resp = await apiCall<WhitelistIP>(`/api/block_list/whitelist/ip`, 'POST', { address }, abortController);

    return transformTimeStampedDates(resp);
};

export const apiUpdateWhitelistIP = async (
    cidr_block_whitelist_id: number,
    address: string,
    abortController?: AbortController
) => {
    const resp = await apiCall<WhitelistIP>(
        `/api/block_list/whitelist/ip/${cidr_block_whitelist_id}`,
        'POST',
        { address },
        abortController
    );

    return transformTimeStampedDates(resp);
};

export const apiDeleteCIDRBlockWhitelist = async (
    cidr_block_whitelist_id: number,
    abortController?: AbortController
) => {
    return await apiCall<EmptyBody>(
        `/api/block_list/whitelist/ip/${cidr_block_whitelist_id}`,
        'DELETE',
        undefined,
        abortController
    );
};

export interface CIDRBlockCheckResponse {
    blocked: boolean;
    source: string;
}

export const apiCIDRBlockCheck = async (address: string, abortController?: AbortController) => {
    return await apiCall<CIDRBlockCheckResponse>(`/api/block_list/checker`, 'POST', { address }, abortController);
};
