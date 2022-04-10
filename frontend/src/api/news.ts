import { apiCall } from './common';

export interface NewsEntry {
    news_id: number;
    title: string;
    body_md: string;
    is_published: boolean;
    created_on: Date;
    updated_on: Date;
}

export const apiGetNewsLatest = async (): Promise<NewsEntry[]> => {
    return await apiCall<NewsEntry[]>(`/api/news_latest`, 'GET');
};
