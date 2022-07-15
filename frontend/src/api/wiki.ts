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

export const apiGetWikiPage = async (slug: string) =>
    await apiCall<Page>(`/api/wiki/slug/${slug}`, 'GET');

export const apiSaveWikiPage = async (page: Page) =>
    await apiCall<Page>(`/api/wiki/slug`, 'POST', page);

export const renderMarkdown = (md: string) =>
    marked(
        md
            .replace(/(wiki:\/\/)/gi, '/wiki/')
            .replace(/(media:\/\/)/gi, '/media/')
    );
