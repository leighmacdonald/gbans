import React, {
    createContext,
    Dispatch,
    SetStateAction,
    useContext,
    useState,
    JSX
} from 'react';
import { noop } from 'lodash-es';
import { UserNotification } from '../api';

export type NotificationState = {
    notifications: UserNotification[];
    selectedIds: number[];
    setSelectedIds: Dispatch<SetStateAction<number[]>>;
};

export const NotificationsCtx = createContext<NotificationState>({
    notifications: [],
    selectedIds: [],
    setSelectedIds: () => noop
});

export const NotificationsProvider = ({
    children
}: {
    children: JSX.Element;
}) => {
    const [selectedIds, setSelectedIds] = useState<number[]>([]);

    const { notifications } = useNotificationsCtx();

    return (
        <NotificationsCtx.Provider
            value={{
                notifications,
                selectedIds,
                setSelectedIds
            }}
        >
            {children}
        </NotificationsCtx.Provider>
    );
};

export const useNotificationsCtx = () => {
    const context = useContext(NotificationsCtx);
    if (context === undefined) {
        throw new Error(
            'useNotifications must be used within a NotificationsProvider'
        );
    }
    return context;
};
