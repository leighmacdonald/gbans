import { LazyResult } from '../component/LazyTableSimple';
import {
    apiCall,
    QueryFilter,
    TimeStamped,
    transformTimeStampedDates
} from './common';

export interface Forum extends TimeStamped {
    forum_id: number;
    forum_category_id: number;
    last_thread_id: number;
    title: string;
    description: string;
    ordering: number;
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
    ordering: number,
    abortContriller?: AbortController
) => {
    return await apiCall<ForumCategory>(
        `/api/forum/category`,
        'POST',
        {
            title,
            description,
            ordering
        },
        abortContriller
    );
};

export const apiCreateForum = async (
    forum_category_id: number,
    title: string,
    description: string,
    ordering: number,
    abortContriller?: AbortController
) => {
    return await apiCall<Forum>(
        `/api/forum/forum`,
        'POST',
        {
            forum_category_id,
            title,
            description,
            ordering
        },
        abortContriller
    );
};

export const apiForum = async (
    forum_id: number,
    abortContriller?: AbortController
) => {
    return await apiCall<Forum>(
        `/api/forum/forum/${forum_id}`,
        'GET',
        undefined,
        abortContriller
    );
};

export const apiSaveForum = async (
    forum_id: number,
    forum_category_id: number,
    title: string,
    description: string,
    ordering: number,
    abortContriller?: AbortController
) => {
    return await apiCall<Forum>(
        `/api/forum/forum/${forum_id}`,
        'POST',
        {
            forum_category_id,
            title,
            description,
            ordering
        },
        abortContriller
    );
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
        return transformTimeStampedDates(c).forums.map(
            transformTimeStampedDates
        );
    });
    return resp;
};

export interface ForumThread extends TimeStamped {
    forum_thread_id: number;
    forum_id: number;
    source_id: string;
    title: string;
    sticky: boolean;
    locked: boolean;
    views: number;
    replies: number;
    personaname: string;
    avatar_hash: string;
}

export interface ThreadQueryOpts extends QueryFilter<ForumThread> {
    forum_id: number;
}

export const apiGetThreads = async (
    opts: ThreadQueryOpts,
    abortController?: AbortController
) => {
    const resp = await apiCall<LazyResult<ForumThread>>(
        `/api/forum/threads`,
        'POST',
        opts,
        abortController
    );
    resp.data = resp.data.map(transformTimeStampedDates);
    return resp;
};
