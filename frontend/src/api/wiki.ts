import { apiCall } from './common';
import { marked, Renderer } from 'marked';

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

class WikiRenderer extends Renderer {
    link(href: string, title: string, text: string) {
        // href = cleanUrl(this.options.sanitize, this.options.baseUrl, href);
        if (href === null) {
            return text;
        }
        if (
            !(
                href.toLowerCase().startsWith('http://') ||
                href.toLowerCase().startsWith('https://')
            )
        ) {
            // noinspection JSDeprecatedSymbols
            href = escape(href);
        }
        let out = '<a href="' + href + '"';
        if (title) {
            out += ' title="' + title + '"';
        }
        out += '>' + text + '</a>';
        return out;
    }
}

export const renderMarkdown = (md: string) =>
    marked(
        md
            .replace(/(wiki:\/\/)/gi, '/wiki/')
            .replace(/(media:\/\/)/gi, '/media/'),
        { renderer: new WikiRenderer() }
    );
