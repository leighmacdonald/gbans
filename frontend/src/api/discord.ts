import { apiCall, CallbackLink } from './common.ts';

export const apiGetDiscordLogin = async () => {
    return apiCall<CallbackLink>('/api/discord/login');
};
