import { apiCall } from './common';
import { marked } from 'marked';
import { BaseUploadedMedia, UserUploadedFile } from './report';

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

export const renderWiki = (md: string) =>
    marked(
        md
            .replace(/(wiki:\/\/)/gi, '/wiki/')
            .replace(/(media:\/\/)/gi, '/wiki_media/')
    );

export interface MediaUploadRequest extends UserUploadedFile {
    wiki_url: string;
}

export interface MediaUploadResponse extends BaseUploadedMedia {
    url: string;
}

export const apiSaveWikiMedia = async (upload: UserUploadedFile) =>
    await apiCall<MediaUploadResponse, UserUploadedFile>(
        `/api/wiki/media`,
        'POST',
        upload
    );
