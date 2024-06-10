import { apiGetBansSteam } from './bans.ts';

export const steamBansQuery = () => {
    return {
        queryKey: ['steamBans'],
        queryFn: async () => {
            return await apiGetBansSteam({
                deleted: false
            });
        }
    };
};
