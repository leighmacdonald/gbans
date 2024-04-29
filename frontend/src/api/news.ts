import { apiCall, TimeStamped, transformTimeStampedDatesList, ValidationException } from './common';

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

export const apiNewsSave = async (entry: NewsEntry) => {
    if (entry.body_md === '') {
        throw new ValidationException(`body_md cannot be empty`);
    }
    if (entry.title === '') {
        throw new ValidationException(`title cannot be empty`);
    }
    if (entry.news_id > 0) {
        return await apiCall<NewsEntry, NewsEntry>(`/api/news/${entry.news_id}`, 'POST', entry);
    } else {
        return await apiCall<NewsEntry, NewsEntry>(`/api/news`, 'POST', entry);
    }
};
