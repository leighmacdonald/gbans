import { apiCall, EmptyBody, TimeStamped, transformTimeStampedDates, transformTimeStampedDatesList } from './common';

export interface CIDRBlockSource extends TimeStamped {
    cidr_block_source_id: number;
    name: string;
    url: string;
    enabled: boolean;
}

export interface CIDRBlockLists {
    sources: CIDRBlockSource[];
    whitelist: CIDRBlockWhitelist[];
}

export const apiGetCIDRBlockLists = async (abortController?: AbortController) => {
    const resp = await apiCall<CIDRBlockLists>(`/api/block_list`, 'GET', undefined, abortController);

    resp.sources = transformTimeStampedDatesList(resp.sources);
    resp.whitelist = transformTimeStampedDatesList(resp.whitelist);

    return resp;
};

export const apiCreateCIDRBlockSource = async (
    name: string,
    url: string,
    enabled: boolean,
    abortController?: AbortController
) => {
    const resp = await apiCall<CIDRBlockSource>(`/api/block_list`, 'POST', { name, url, enabled }, abortController);
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
        `/api/block_list/${cidr_block_source_id}`,
        'POST',
        { name, url, enabled },
        abortController
    );
    return transformTimeStampedDates(resp);
};

export const apiDeleteCIDRBlockSource = async (cidr_block_source_id: number, abortController?: AbortController) => {
    return await apiCall<EmptyBody>(`/api/block_list/${cidr_block_source_id}`, 'DELETE', undefined, abortController);
};

export interface CIDRBlockWhitelist extends TimeStamped {
    cidr_block_whitelist_id: number;
    address: string;
}

export const apiCreateCIDRBlockWhitelist = async (address: string, abortController?: AbortController) => {
    const resp = await apiCall<CIDRBlockWhitelist>(`/api/block_list/whitelist`, 'POST', { address }, abortController);

    return transformTimeStampedDates(resp);
};

export const apiUpdateCIDRBlockWhitelist = async (
    cidr_block_whitelist_id: number,
    address: string,
    abortController?: AbortController
) => {
    const resp = await apiCall<CIDRBlockWhitelist>(
        `/api/block_list/whitelist/${cidr_block_whitelist_id}`,
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
        `/api/block_list/whitelist/${cidr_block_whitelist_id}`,
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
