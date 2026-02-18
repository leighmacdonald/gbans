import type { ChatStatus, OperationEnum } from "../schema/playerqueue.ts";
import { apiCall } from "./common.ts";

export const apiQueueMessagesDelete = async (message_id: number, count: number) => {
	return await apiCall(`/api/playerqueue/messages/${message_id}/${count}`, "DELETE", {});
};

export const apiQueueSetUserStatus = async (steam_id: string, chat_status: ChatStatus, reason: string) => {
	return await apiCall(`/api/playerqueue/status/${steam_id}`, "PUT", {
		chat_status,
		reason,
	});
};

export type QueueRequest<T> = {
	op: OperationEnum;
	payload: T;
};

export const websocketURL = () => {
	let protocol = "ws";
	if (location.protocol === "https:") {
		protocol = "wss";
	}
	return `${protocol}://${location.host}/ws`;
};
