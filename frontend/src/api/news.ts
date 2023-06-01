import { apiCall, ValidationException } from './common';

export interface NewsEntry {
    news_id: number;
    title: string;
    body_md: string;
    is_published: boolean;
    created_on?: string;
    updated_on?: string;
}

export const apiGetNewsLatest = async () =>
    await apiCall<NewsEntry[]>(`/api/news_latest`, 'POST');

export const apiGetNewsAll = async () =>
    await apiCall<NewsEntry[]>(`/api/news_all`, 'POST');

export const apiNewsSave = async (entry: NewsEntry) => {
    if (entry.body_md === '') {
        throw new ValidationException(`body_md cannot be empty`);
    }
    if (entry.title === '') {
        throw new ValidationException(`title cannot be empty`);
    }
    if (entry.news_id > 0) {
        return await apiCall<NewsEntry, NewsEntry>(
            `/api/news/${entry.news_id}`,
            'POST',
            entry
        );
    } else {
        return await apiCall<NewsEntry, NewsEntry>(`/api/news`, 'POST', entry);
    }
};
