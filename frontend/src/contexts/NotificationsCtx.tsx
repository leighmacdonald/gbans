import React, {
    createContext,
    Dispatch,
    SetStateAction,
    useContext,
    useState,
    JSX,
    useEffect
} from 'react';
import { noop } from 'lodash-es';
import {
    apiGetNotifications,
    NotificationsQuery,
    UserNotification
} from '../api';
import { useCurrentUserCtx } from './CurrentUserCtx';
import { logErr } from '../util/errors';

export type NotificationState = {
    notifications: UserNotification[];
    setNotifications: Dispatch<SetStateAction<UserNotification[]>>;
    selectedIds: number[];
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
    const [notifications, setNotifications] = useState<UserNotification[]>([]);
    const [selectedIds, setSelectedIds] = useState<number[]>([]);
    // NOTE: you *might* need to memoize this value
    // Learn more in http://kcd.im/optimize-context
    const { currentUser } = useCurrentUserCtx();

    useEffect(() => {
        if (currentUser.steam_id) {
            const query: NotificationsQuery = {};
            apiGetNotifications(query)
                .then((res) => {
                    setNotifications(res);
                })
                .catch(logErr);
        }
    }, [currentUser.steam_id, notifications.length]);

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

export const useNotifications = () => {
    const context = useContext(NotificationsCtx);
    if (context === undefined) {
        throw new Error(
            'useNotifications must be used within a NotificationsProvider'
        );
    }
    return context;
};
