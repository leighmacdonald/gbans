import { useContext } from 'react';
import { NotificationsCtx } from '../contexts/NotificationsCtx.tsx';

export const useNotificationsCtx = () => {
    const context = useContext(NotificationsCtx);
    if (context === undefined) {
        throw new Error(
            'useNotifications must be used within a NotificationsProvider'
        );
    }
    return context;
};
