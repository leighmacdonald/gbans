import { createContext, type Dispatch, type SetStateAction } from "react";
import type { UserNotification } from "../schema/people.ts";
import { noop } from "../util/lists.ts";

export type NotificationState = {
	notifications: UserNotification[];
	selectedIds: number[];
	setNotifications: Dispatch<SetStateAction<UserNotification[]>>;
	setSelectedIds: Dispatch<SetStateAction<number[]>>;
};

export const NotificationsCtx = createContext<NotificationState>({
	notifications: [],
	setNotifications: () => noop,
	selectedIds: [],
	setSelectedIds: () => noop,
});
