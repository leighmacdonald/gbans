import { apiCall, PermissionLevel, TimeStamped } from './common';

export interface Page extends TimeStamped {
    slug: string;
    title: string;
    body_md: string;
    revision: number;
    permission_level: PermissionLevel;
}

export const apiGetWikiPage = async (slug: string, abortController?: AbortController) =>
    await apiCall<Page>(`/api/wiki/slug/${slug}`, 'GET', undefined, abortController);

export const apiSaveWikiPage = async (page: Page) => await apiCall<Page>(`/api/wiki/slug`, 'POST', page);
