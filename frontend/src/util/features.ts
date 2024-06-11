import { redirect } from '@tanstack/react-router';
import { appInfoDetail } from '../api/app.ts';

export const checkFeatureEnabled = (featureName: keyof appInfoDetail, redirectTo: string = '/') => {
    const item = localStorage.getItem('appInfo');
    if (item) {
        const appInfo = JSON.parse(item) as appInfoDetail;
        if (!appInfo[featureName]) {
            throw redirect({
                to: redirectTo
            });
        }
    }
};
