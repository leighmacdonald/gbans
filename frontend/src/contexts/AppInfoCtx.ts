import { createContext, useContext } from 'react';
import { appInfoDetail } from '../api';
import { noop } from '../util/lists.ts';

export type AppInfoCtx = {
    appInfo: appInfoDetail;
    setAppInfo: (appInfo: appInfoDetail) => void;
};

export const UseAppInfoCtx = createContext<AppInfoCtx>({
    setAppInfo: () => noop,
    appInfo: {
        app_version: 'master',
        link_id: '',
        sentry_dns_web: '',
        site_name: 'Loading',
        asset_url: '/assets',
        patreon_client_id: '',
        discord_client_id: '',
        patreon_enabled: false,
        discord_enabled: false
    }
});

export const useAppInfoCtx = () => useContext(UseAppInfoCtx);
