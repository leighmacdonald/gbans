import {
    ConnectionQuery,
    DiscordUser,
    IPQuery,
    MessageQuery,
    NetworkDetails,
    PermissionUpdate,
    Person,
    PersonConnection,
    PersonMessage,
    PersonSettings,
    PlayerProfile,
    PlayerQuery,
    SteamValidate,
    UserNotification,
    UserProfile
} from '../schema/people.ts';
import { LazyResult } from '../util/table.ts';
import { parseDateTime, transformCreatedOnDate, transformTimeStampedDates } from '../util/time.ts';
import { apiCall } from './common';

export const apiGetSteamValidate = async (query: string) => {
    try {
        return await apiCall<SteamValidate>(`/api/steam/validate?query=${query}`);
    } catch {
        return {
            hash: '',
            personaname: '',
            steam_id: ''
        } as SteamValidate;
    }
};

export const apiGetProfile = async (query: string, abortController?: AbortController) => {
    const profile = await apiCall<PlayerProfile>(`/api/profile?query=${query}`, 'GET', undefined, abortController);
    profile.player = transformTimeStampedDates(profile.player);
    return profile;
};

export const apiGetCurrentProfile = async () => apiCall<UserProfile>(`/api/current_profile`, 'GET', undefined);

export const apiSearchPeople = async (opts: PlayerQuery, abortController?: AbortController) => {
    const people = await apiCall<LazyResult<Person>>(`/api/players`, 'POST', opts, abortController);
    people.data = people.data.map(transformTimeStampedDates);
    return people;
};

export const apiGetMessageContext = async (messageId: number, padding: number = 5) => {
    const resp = await apiCall<PersonMessage[]>(`/api/message/${messageId}/context/${padding}`, 'GET');
    return resp.map((msg) => {
        return {
            ...msg,
            created_on: parseDateTime(msg.created_on as unknown as string)
        };
    });
};

export const apiGetMessages = async (opts: MessageQuery, abortController?: AbortController) => {
    const resp = await apiCall<PersonMessage[], MessageQuery>(
        `/api/messages`,
        'POST',
        {
            ...opts,
            date_start: (opts.date_start ?? '') == '' ? undefined : opts.date_start,
            date_end: (opts.date_end ?? '') == '' ? undefined : opts.date_end
        },
        abortController
    );

    return resp.map((msg) => {
        return {
            ...msg,
            created_on: parseDateTime(msg.created_on as unknown as string)
        };
    });
};

export const apiGetNotifications = async (abortController?: AbortController) => {
    const resp = await apiCall<UserNotification[]>(`/api/notifications`, 'GET', undefined, abortController);
    return resp.map((msg) => {
        return { ...msg, created_on: parseDateTime(msg.created_on as unknown as string) };
    });
};

export const apiNotificationsMarkAllRead = async () => {
    return await apiCall(`/api/notifications/all`, 'POST', undefined);
};

export const apiNotificationsMarkRead = async (message_ids: number[]) => {
    return await apiCall(`/api/notifications`, 'POST', { message_ids });
};

export const apiNotificationsDeleteAll = async () => {
    return await apiCall(`/api/notifications/all`, 'DELETE', undefined);
};

export const apiNotificationsDelete = async (message_ids: number[]) => {
    return await apiCall(`/api/notifications`, 'DELETE', { message_ids });
};

export const apiGetConnections = async (opts: ConnectionQuery, abortController?: AbortController) => {
    const resp = await apiCall<LazyResult<PersonConnection>, ConnectionQuery>(
        `/api/connections`,
        'POST',
        opts,
        abortController
    );

    resp.data = resp.data.map(transformCreatedOnDate);
    return resp;
};

export const apiGetNetworkDetails = async (opts: IPQuery, abortController?: AbortController) => {
    return await apiCall<NetworkDetails, IPQuery>(`/api/network`, 'POST', opts, abortController);
};

export const apiUpdatePlayerPermission = async (
    steamId: string,
    args: PermissionUpdate,
    abortController?: AbortController
) =>
    transformTimeStampedDates(
        await apiCall<Person, PermissionUpdate>(`/api/player/${steamId}/permissions`, 'PUT', args, abortController)
    );

export const apiGetPersonSettings = async (abortController?: AbortController) => {
    return transformTimeStampedDates(
        await apiCall<PersonSettings>(`/api/current_profile/settings`, 'GET', undefined, abortController)
    );
};

export const apiSavePersonSettings = async (
    forum_signature: string,
    forum_profile_messages: boolean,
    stats_hidden: boolean,
    center_projectiles: boolean,
    abortController?: AbortController
) => {
    return transformTimeStampedDates(
        await apiCall<PersonSettings>(
            `/api/current_profile/settings`,
            'POST',
            { forum_signature, forum_profile_messages, stats_hidden, center_projectiles },
            abortController
        )
    );
};

export const apiDiscordUser = async () => {
    return transformTimeStampedDates(await apiCall<DiscordUser>('/api/discord/user'));
};

export const discordAvatarURL = (user: DiscordUser) => {
    return `https://cdn.discordapp.com/avatars/${user.id}/${user.avatar}.png`;
};

export const apiDiscordLogout = async () => {
    return await apiCall<DiscordUser>('/api/discord/logout');
};
