import { JSX, useState } from 'react';
import { UserNotification } from '../api';
import { NotificationsCtx } from '../contexts/NotificationsCtx.tsx';

export const NotificationsProvider = ({ children }: { children: JSX.Element }) => {
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
