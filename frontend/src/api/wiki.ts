import { apiCall } from './common';
import { marked } from 'marked';

export interface Page {
    slug: string;
    title: string;
    body_md: string;
    revision: number;
    created_on: Date;
    updated_on: Date;
}

export const apiGetWikiPage = async (slug: string): Promise<Page> => {
    return await apiCall<Page>(`/api/wiki/slug/${slug}`, 'GET');
};

export const apiSaveWikiPage = async (page: Page): Promise<Page> => {
    return await apiCall<Page>(`/api/wiki/slug`, 'POST', page);
};

export const renderWiki = (md: string) => {
    md = md.replace(/(wiki:\/\/)/gi, '/wiki/');
    return marked(md);
};
