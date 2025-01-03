import { createContext, Dispatch, SetStateAction } from 'react';
import { UserNotification } from '../api';
import { noop } from '../util/lists.ts';

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
    setSelectedIds: () => noop
});
