import { z } from 'zod';
import { schemaTimeStamped } from './chrono.ts';
import { PermissionLevelEnum } from './people.ts';

export const schemaForum = z
    .object({
        forum_id: z.number(),
        forum_category_id: z.number(),
        last_thread_id: z.number(),
        title: z.string(),
        description: z.string(),
        ordering: z.number(),
        count_threads: z.number(),
        count_messages: z.number(),
        permission_level: PermissionLevelEnum,
        recent_forum_thread_id: z.number().optional(),
        recent_forum_title: z.string().optional(),
        recent_source_id: z.string().optional(),
        recent_avatarhash: z.string().optional(),
        recent_personaname: z.string().optional(),
        recent_created_on: z.date().optional()
    })
    .merge(schemaTimeStamped);

export type Forum = z.infer<typeof schemaForum>;

export const schemaForumCategory = z
    .object({
        forum_category_id: z.number(),
        title: z.string(),
        description: z.string(),
        ordering: z.number(),
        forums: z.array(schemaForum)
    })
    .merge(schemaTimeStamped);

export type ForumCategory = z.infer<typeof schemaForumCategory>;

export const schemaForumMessage = z
    .object({
        forum_message_id: z.number(),
        forum_thread_id: z.number(),
        source_id: z.string(),
        body_md: z.string(),
        personaname: z.string(),
        avatarhash: z.string(),
        online: z.boolean(),
        title: z.string(),
        permission_level: PermissionLevelEnum,
        signature: z.string()
    })
    .merge(schemaTimeStamped);

export type ForumMessage = z.infer<typeof schemaForumMessage>;

export const schemaForumThread = z
    .object({
        forum_thread_id: z.number(),
        forum_id: z.number(),
        source_id: z.string(),
        title: z.string(),
        sticky: z.boolean(),
        locked: z.boolean(),
        views: z.number(),
        replies: z.number(),
        personaname: z.string(),
        avatarhash: z.string(),
        message: schemaForumMessage.optional(),
        recent_forum_message_id: z.number().optional(),
        recent_created_on: z.date(),
        recent_steam_id: z.string(),
        recent_personaname: z.string(),
        recent_avatarhash: z.string()
    })
    .merge(schemaTimeStamped);

export type ForumThread = z.infer<typeof schemaForumThread>;

export const schemaThreadMessageQueryOpts = z.object({
    forum_thread_id: z.number(),
    deleted: z.boolean().optional()
});

export type ThreadMessageQueryOpts = z.infer<typeof schemaThreadMessageQueryOpts>;

export const schemaActiveUser = z.object({
    steam_id: z.string(),
    personaname: z.string(),
    permission_level: PermissionLevelEnum,
    created_on: z.date()
});

export type ActiveUser = z.infer<typeof schemaActiveUser>;

export const schemaForumOverview = z.object({
    categories: z.array(schemaForumCategory)
});

export type ForumOverview = z.infer<typeof schemaForumOverview>;
