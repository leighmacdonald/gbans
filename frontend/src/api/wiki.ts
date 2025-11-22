import { Page } from '../schema/wiki.ts';
import { apiCall } from './common';

export const apiGetWikiPage = async (slug: string, abortController?: AbortController) =>
    await apiCall<Page>(`/api/wiki/slug/${slug}`, 'GET', undefined, abortController);

export const apiSaveWikiPage = async (page: Page) => await apiCall<Page>(`/api/wiki/slug/${page.slug}`, 'PUT', page);