import { createContext, Dispatch, SetStateAction, useState, JSX } from 'react';
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

export const NotificationsProvider = ({
    children
}: {
    children: JSX.Element;
}) => {
    const [selectedIds, setSelectedIds] = useState<number[]>([]);
    const [notifications, setNotifications] = useState<UserNotification[]>([]);

    return (
        <NotificationsCtx.Provider
            value={{
                setNotifications,
                notifications,
                selectedIds,
                setSelectedIds
            }}
        >
            {children}
        </NotificationsCtx.Provider>
    );
};
