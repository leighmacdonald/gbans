import type {
	ActiveUser,
	Forum,
	ForumCategory,
	ForumMessage,
	ForumOverview,
	ForumThread,
	ThreadMessageQueryOpts,
} from "../schema/forum.ts";
import type { PermissionLevelEnum } from "../schema/people.ts";
import { parseDateTime, transformCreatedOnDate, transformTimeStampedDates } from "../util/time.ts";
import { apiCall } from "./common";

export const apiGetForumCategory = async (forumCategoryId: number, signal: AbortSignal) => {
	return transformTimeStampedDates(await apiCall<ForumCategory>(signal, `/api/forum/category/${forumCategoryId}`));
};

export const apiSaveForumCategory = async (
	forum_category_id: number,
	title: string,
	description: string,
	ordering: number,
	signal: AbortSignal,
) => {
	return await apiCall<ForumCategory>(signal, `/api/forum/category/${forum_category_id}`, "POST", {
		title,
		description,
		ordering,
	});
};

export const apiCreateForumCategory = async (
	title: string,
	description: string,
	ordering: number,
	signal: AbortSignal,
) => {
	return await apiCall<ForumCategory>(signal, `/api/forum/category`, "POST", {
		title,
		description,
		ordering,
	});
};

export const apiCreateForum = async (
	forum_category_id: number,
	title: string,
	description: string,
	ordering: number,
	permission_level: PermissionLevelEnum,
	signal: AbortSignal,
) => {
	return await apiCall<Forum>(signal, `/api/forum/forum`, "POST", {
		forum_category_id,
		title,
		description,
		ordering,
		permission_level,
	});
};

export const apiForum = async (forum_id: number, signal: AbortSignal) => {
	return await apiCall<Forum>(signal, `/api/forum/forum/${forum_id}`);
};

export const apiSaveForum = async (
	forum_id: number,
	forum_category_id: number,
	title: string,
	description: string,
	ordering: number,
	permission_level: PermissionLevelEnum,
	signal: AbortSignal,
) => {
	return await apiCall<Forum>(signal, `/api/forum/forum/${forum_id}`, "POST", {
		forum_category_id,
		title,
		description,
		ordering,
		permission_level,
	});
};

export const apiGetForumOverview = async (signal: AbortSignal) => {
	const resp = await apiCall<ForumOverview>(signal, "/api/forum/overview");
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

export const apiGetThreadMessages = async (opts: ThreadMessageQueryOpts, signal: AbortSignal) => {
	const resp = await apiCall<ForumMessage[]>(signal, `/api/forum/messages`, "POST", opts);
	return resp.map(transformTimeStampedDates);
};

export const apiSaveThreadMessage = async (forum_message_id: number, body_md: string, signal: AbortSignal) => {
	const resp = await apiCall<ForumMessage>(signal, `/api/forum/message/${forum_message_id}`, "POST", { body_md });
	return transformTimeStampedDates(resp);
};

export interface ThreadQueryOpts {
	forum_id: number;
}

export const apiGetThreads = async (opts: ThreadQueryOpts, signal: AbortSignal) => {
	const resp = await apiCall<ForumThread[]>(signal, `/api/forum/threads`, "POST", opts);
	if (!resp) {
		return [];
	}
	return resp.map((t) => {
		const thread = transformTimeStampedDates(t);
		thread.recent_created_on = parseDateTime(thread.recent_created_on as unknown as string);
		return thread;
	});
};

export const apiGetThread = async (thread_id: number, signal: AbortSignal) => {
	const resp = await apiCall<ForumThread>(signal, `/api/forum/thread/${thread_id}`);
	return transformTimeStampedDates(resp);
};

export const apiDeleteThread = async (thread_id: number, signal: AbortSignal) => {
	return await apiCall(signal, `/api/forum/thread/${thread_id}`, "DELETE");
};

export const apiUpdateThread = async (
	thread_id: number,
	title: string,
	sticky: boolean,
	locked: boolean,
	signal: AbortSignal,
) => {
	const resp = await apiCall<ForumThread>(signal, `/api/forum/thread/${thread_id}`, "POST", {
		title,
		sticky,
		locked,
	});
	return transformTimeStampedDates(resp);
};

export const apiCreateThread = async (
	forum_id: number,
	title: string,
	body_md: string,
	sticky: boolean,
	locked: boolean,
	signal: AbortSignal,
) => {
	const resp = await apiCall<ForumThread>(signal, `/api/forum/forum/${forum_id}/thread`, "POST", {
		title,
		body_md,
		sticky,
		locked,
	});
	return transformTimeStampedDates(resp);
};

export const apiCreateThreadReply = async (forum_thread_id: number, body_md: string, signal: AbortSignal) => {
	return transformTimeStampedDates(
		await apiCall<ForumMessage>(signal, `/api/forum/thread/${forum_thread_id}/message`, "POST", { body_md }),
	);
};

export const apiDeleteMessage = async (forum_message_id: number, signal: AbortSignal) => {
	return await apiCall(signal, `/api/forum/message/${forum_message_id}`, "DELETE");
};

export const apiForumRecentActivity = async (signal: AbortSignal) => {
	return (await apiCall<ForumMessage[]>(signal, `/api/forum/messages/recent`)).map(transformTimeStampedDates);
};

export const apiForumActiveUsers = async (signal: AbortSignal) => {
	const resp = await apiCall<ActiveUser[]>(signal, `/api/forum/active_users`);
	return resp.map(transformCreatedOnDate);
};
