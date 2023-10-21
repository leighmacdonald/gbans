import { apiCall, TimeStamped } from './common';
import { marked, Renderer } from 'marked';
import { gfmHeadingId } from 'marked-gfm-heading-id';
import { mangle } from 'marked-mangle';

export interface Page extends TimeStamped {
    slug: string;
    title: string;
    body_md: string;
    revision: number;
}

export const apiGetWikiPage = async (
    slug: string,
    abortController?: AbortController
) =>
    await apiCall<Page>(
        `/api/wiki/slug/${slug}`,
        'GET',
        undefined,
        abortController
    );

export const apiSaveWikiPage = async (page: Page) =>
    await apiCall<Page>(`/api/wiki/slug`, 'POST', page);

// escape() replacement
const fixedEncodeURI = (str: string) =>
    encodeURI(str).replace(/%5B/g, '[').replace(/%5D/g, ']');

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
            href = fixedEncodeURI(href);
        }
        let out = '<a href="' + href + '"';
        if (title) {
            out += ' title="' + title + '"';
        }
        out += '>' + text + '</a>';
        return out;
    }
}

const options = {
    prefix: 'gb-'
};

marked.use(gfmHeadingId(options), mangle());

export const renderMarkdown = (md: string) =>
    marked(
        md
            // special ZERO WIDTH unicode characters (for example \uFEFF) might interfere with parsing
            .replace('/^[\u200B\u200C\u200D\u200E\u200F\uFEFF]/', '')
            .replace(/(wiki:\/\/)/gi, '/wiki/')
            .replace(
                /(media:\/\/)/gi,
                window.gbans.asset_url + '/' + window.gbans.bucket_media + '/'
            ),
        { renderer: new WikiRenderer(), gfm: true, async: true }
    );
