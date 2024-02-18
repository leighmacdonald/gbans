import { LazyResult } from '../component/table/LazyTableSimple';
import { parseDateTime } from '../util/text.tsx';
import {
    apiCall,
    PermissionLevel,
    QueryFilter,
    TimeStamped,
    transformCreatedOnDate,
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
    permission_level: PermissionLevel;
    recent_forum_thread_id?: number;
    recent_forum_title?: string;
    recent_source_id?: string;
    recent_avatarhash?: string;
    recent_personaname?: string;
    recent_created_on?: Date;
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
    permission_level: PermissionLevel,
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

export const apiForum = async (
    forum_id: number,
    abortController?: AbortController
) => {
    return await apiCall<Forum>(
        `/api/forum/forum/${forum_id}`,
        'GET',
        undefined,
        abortController
    );
};

export const apiSaveForum = async (
    forum_id: number,
    forum_category_id: number,
    title: string,
    description: string,
    ordering: number,
    permission_level: PermissionLevel,
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
    resp.categories = resp.categories.map((category) => {
        const cat = transformTimeStampedDates(category);
        cat.forums = cat.forums.map((forum) => {
            const f = transformTimeStampedDates(forum);
            if (f.recent_created_on) {
                f.recent_created_on = parseDateTime(
                    f.recent_created_on as unknown as string
                );
            }
            return f;
        });
        return cat;
    });

    return resp;
};

export interface ForumMessage extends TimeStamped {
    forum_message_id: number;
    forum_thread_id: number;
    source_id: string;
    body_md: string;
    personaname: string;
    avatarhash: string;
    online: boolean;
    title: string;
    permission_level: PermissionLevel;
    signature: string;
}

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
    avatarhash: string;
    message?: ForumMessage;
    recent_forum_message_id?: number;
    recent_created_on: Date;
    recent_steam_id: string;
    recent_personaname: string;
    recent_avatarhash: string;
}

export interface ThreadMessageQueryOpts extends QueryFilter<ForumMessage> {
    forum_thread_id: number;
}

export const apiGetThreadMessages = async (
    opts: ThreadMessageQueryOpts,
    abortController?: AbortController
) => {
    const resp = await apiCall<LazyResult<ForumMessage>>(
        `/api/forum/messages`,
        'POST',
        opts,
        abortController
    );
    resp.data = resp.data.map(transformTimeStampedDates);
    return resp;
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
    resp.data = resp.data.map((t) => {
        const thread = transformTimeStampedDates(t);
        thread.recent_created_on = parseDateTime(
            thread.recent_created_on as unknown as string
        );
        return thread;
    });
    return resp;
};

export const apiGetThread = async (
    thread_id: number,
    abortController?: AbortController
) => {
    const resp = await apiCall<ForumThread>(
        `/api/forum/thread/${thread_id}`,
        'GET',
        undefined,
        abortController
    );
    return transformTimeStampedDates(resp);
};

export const apiDeleteThread = async (
    thread_id: number,
    abortController?: AbortController
) => {
    return await apiCall(
        `/api/forum/thread/${thread_id}`,
        'DELETE',
        undefined,
        abortController
    );
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

export const apiDeleteMessage = async (
    forum_message_id: number,
    abortController?: AbortController
) => {
    return await apiCall(
        `/api/forum/message/${forum_message_id}`,
        'DELETE',
        undefined,
        abortController
    );
};

export const apiForumRecentActivity = async (
    abortController?: AbortController
) => {
    return (
        await apiCall<ForumMessage[]>(
            `/api/forum/messages/recent`,
            'GET',
            undefined,
            abortController
        )
    ).map(transformTimeStampedDates);
};

export interface ActiveUser {
    steam_id: string;
    personaname: string;
    permission_level: PermissionLevel;
    created_on: Date;
}

export const apiForumActiveUsers = async (
    abortController?: AbortController
) => {
    const resp = await apiCall<ActiveUser[]>(
        `/api/forum/active_users`,
        'GET',
        undefined,
        abortController
    );
    return resp.map(transformCreatedOnDate);
};
