import {
    ActiveUser,
    Forum,
    ForumCategory,
    ForumMessage,
    ForumOverview,
    ForumThread,
    ThreadMessageQueryOpts
} from '../schema/forum.ts';
import { PermissionLevelEnum } from '../schema/people.ts';
import { parseDateTime, transformCreatedOnDate, transformTimeStampedDates } from '../util/time.ts';
import { apiCall } from './common';

export const apiGetForumCategory = async (forumCategoryId: number, abortController?: AbortController) => {
    return transformTimeStampedDates(
        await apiCall<ForumCategory>(`/api/forum/category/${forumCategoryId}`, 'GET', undefined, abortController)
    );
};

export const apiSaveForumCategory = async (
    forum_category_id: number,
    title: string,
    description: string,
    ordering: number
) => {
    return await apiCall<ForumCategory>(`/api/forum/category/${forum_category_id}`, 'POST', {
        title,
        description,
        ordering
    });
};

export const apiCreateForumCategory = async (
    title: string,
    description: string,
    ordering: number,
    abortController?: AbortController
) => {
    return await apiCall<ForumCategory>(
        `/api/forum/category`,
        'POST',
        {
            title,
            description,
            ordering
        },
        abortController
    );
};

export const apiCreateForum = async (
    forum_category_id: number,
    title: string,
    description: string,
    ordering: number,
    permission_level: PermissionLevelEnum,
    abortController?: AbortController
) => {
    return await apiCall<Forum>(
        `/api/forum/forum`,
        'POST',
        {
            forum_category_id,
            title,
            description,
            ordering,
            permission_level
        },
        abortController
    );
};

export const apiForum = async (forum_id: number, abortController?: AbortController) => {
    return await apiCall<Forum>(`/api/forum/forum/${forum_id}`, 'GET', undefined, abortController);
};

export const apiSaveForum = async (
    forum_id: number,
    forum_category_id: number,
    title: string,
    description: string,
    ordering: number,
    permission_level: PermissionLevelEnum,
    abortController?: AbortController
) => {
    return await apiCall<Forum>(
        `/api/forum/forum/${forum_id}`,
        'POST',
        {
            forum_category_id,
            title,
            description,
            ordering,
            permission_level
        },
        abortController
    );
};

export const apiGetForumOverview = async (abortController?: AbortController) => {
    const resp = await apiCall<ForumOverview>('/api/forum/overview', 'GET', undefined, abortController);
    resp.categories = resp.categories.map((category) => {
        const cat = transformTimeStampedDates(category);
        cat.forums = cat.forums.map((forum) => {
            const f = transformTimeStampedDates(forum);
            if (f.recent_created_on) {
                f.recent_created_on = parseDateTime(f.recent_created_on as unknown as string);
            }
            return f;
        });
        return cat;
    });

    return resp;
};

export const apiGetThreadMessages = async (opts: ThreadMessageQueryOpts, abortController?: AbortController) => {
    const resp = await apiCall<ForumMessage[]>(`/api/forum/messages`, 'POST', opts, abortController);
    return resp.map(transformTimeStampedDates);
};

export const apiSaveThreadMessage = async (
    forum_message_id: number,
    body_md: string,
    abortController?: AbortController
) => {
    const resp = await apiCall<ForumMessage>(
        `/api/forum/message/${forum_message_id}`,
        'POST',
        { body_md },
        abortController
    );
    return transformTimeStampedDates(resp);
};

export interface ThreadQueryOpts {
    forum_id: number;
}

export const apiGetThreads = async (opts: ThreadQueryOpts, abortController?: AbortController) => {
    const resp = await apiCall<ForumThread[]>(`/api/forum/threads`, 'POST', opts, abortController);
    if (!resp) {
        return [];
    }
    return resp.map((t) => {
        const thread = transformTimeStampedDates(t);
        thread.recent_created_on = parseDateTime(thread.recent_created_on as unknown as string);
        return thread;
    });
};

export const apiGetThread = async (thread_id: number, abortController?: AbortController) => {
    const resp = await apiCall<ForumThread>(`/api/forum/thread/${thread_id}`, 'GET', undefined, abortController);
    return transformTimeStampedDates(resp);
};

export const apiDeleteThread = async (thread_id: number, abortController?: AbortController) => {
    return await apiCall(`/api/forum/thread/${thread_id}`, 'DELETE', undefined, abortController);
};

export const apiUpdateThread = async (
    thread_id: number,
    title: string,
    sticky: boolean,
    locked: boolean,
    abortController?: AbortController
) => {
    const resp = await apiCall<ForumThread>(
        `/api/forum/thread/${thread_id}`,
        'POST',
        { title, sticky, locked },
        abortController
    );
    return transformTimeStampedDates(resp);
};

export const apiCreateThread = async (
    forum_id: number,
    title: string,
    body_md: string,
    sticky: boolean,
    locked: boolean,
    abortController?: AbortController
) => {
    const resp = await apiCall<ForumThread>(
        `/api/forum/forum/${forum_id}/thread`,
        'POST',
        { title, body_md, sticky, locked },
        abortController
    );
    return transformTimeStampedDates(resp);
};

export const apiCreateThreadReply = async (
    forum_thread_id: number,
    body_md: string,
    abortController?: AbortController
) => {
    return transformTimeStampedDates(
        await apiCall<ForumMessage>(
            `/api/forum/thread/${forum_thread_id}/message`,
            'POST',
            { body_md },
            abortController
        )
    );
};

export const apiDeleteMessage = async (forum_message_id: number, abortController?: AbortController) => {
    return await apiCall(`/api/forum/message/${forum_message_id}`, 'DELETE', undefined, abortController);
};

export const apiForumRecentActivity = async (abortController?: AbortController) => {
    return (await apiCall<ForumMessage[]>(`/api/forum/messages/recent`, 'GET', undefined, abortController)).map(
        transformTimeStampedDates
    );
};

export const apiForumActiveUsers = async (abortController?: AbortController) => {
    const resp = await apiCall<ActiveUser[]>(`/api/forum/active_users`, 'GET', undefined, abortController);
    return resp.map(transformCreatedOnDate);
};
