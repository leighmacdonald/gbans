import { apiCall, TimeStamped, transformTimeStampedDates } from './common';

export interface Forum extends TimeStamped {
    forum_id: number;
    forum_category_id: number;
    last_thread_id: number;
    title: string;
    description: string;
    ordering: string;
    count_threads: number;
    count_messages: number;
}

export interface ForumCategory extends TimeStamped {
    forum_category_id: number;
    title: string;
    description: string;
    ordering: number;
    forums: Forum[];
}

export const apiGetForumCategory = async (
    forumCategoryId: number,
    abortController?: AbortController
) => {
    return transformTimeStampedDates(
        await apiCall<ForumCategory>(
            `/api/forum/category/${forumCategoryId}`,
            'GET',
            undefined,
            abortController
        )
    );
};

export const apiSaveForumCategory = async (
    forum_category_id: number,
    title: string,
    description: string,
    ordering: number
) => {
    return await apiCall<ForumCategory>(
        `/api/forum/category/${forum_category_id}`,
        'POST',
        { title, description, ordering }
    );
};

export const apiCreateForumCategory = async (
    title: string,
    description: string,
    ordering: number
) => {
    return await apiCall<ForumCategory>(`/api/forum/category`, 'POST', {
        title,
        description,
        ordering
    });
};

export interface ForumOverview {
    categories: ForumCategory[];
}

export const apiGetForumOverview = async (
    abortController?: AbortController
) => {
    const resp = await apiCall<ForumOverview>(
        '/api/forum/overview',
        'GET',
        undefined,
        abortController
    );
    resp.categories.map((c) => {
        const v = transformTimeStampedDates(c);
        v.forums = v.forums.map(transformTimeStampedDates);
        return v;
    });
    return resp;
};
