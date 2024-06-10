import { apiCall, TimeStamped, transformTimeStampedDates, transformTimeStampedDatesList } from './common';

export interface NewsEntry extends TimeStamped {
    news_id: number;
    title: string;
    body_md: string;
    is_published: boolean;
}

export const apiGetNewsLatest = async (abortController?: AbortController) =>
    transformTimeStampedDatesList(await apiCall<NewsEntry[]>(`/api/news_latest`, 'POST', undefined, abortController));

export const apiGetNewsAll = async (abortController?: AbortController) =>
    transformTimeStampedDatesList(await apiCall<NewsEntry[]>(`/api/news_all`, 'POST', undefined, abortController));

export const apiNewsCreate = async (title: string, body_md: string, is_published: boolean) =>
    transformTimeStampedDates(
        await apiCall<NewsEntry>(`/api/news`, 'POST', { title, body_md, is_published: is_published })
    );

export const apiNewsSave = async (entry: NewsEntry) =>
    transformTimeStampedDates(
        await apiCall<NewsEntry>(`/api/news/${entry.news_id}`, 'POST', {
            title: entry.title,
            body_md: entry.body_md,
            is_published: entry.is_published
        })
    );

export const apiNewsDelete = async (news_id: number) => await apiCall<NewsEntry>(`/api/news/${news_id}`, 'DELETE');
