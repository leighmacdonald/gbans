import { formatDistance, parseISO, parseJSON } from 'date-fns';
import format from 'date-fns/format';
import SteamID from 'steamid';
import { apiGetProfile, Person } from '../api';
import { emptyOrNullString } from './types';

export const parseDateTime = (t: string): Date => {
    return parseISO(t);
};

export const renderDateTime = (t: Date): string => {
    return format(t, 'Y-M-d HH:mm');
};

export const renderDate = (t: Date): string => {
    return format(t, 'Y-MM-dd');
};

export const renderTime = (t: Date): string => {
    return format(t, 'HH:mm');
};

export const renderTimeDistance = (
    t1: Date | string,
    t2?: Date | string
): string => {
    if (typeof t1 === 'string') {
        t1 = parseJSON(t1);
    }
    if (!t2) {
        t2 = new Date();
    }
    if (typeof t2 === 'string') {
        t2 = parseJSON(t2);
    }
    return formatDistance(t1, t2, {
        addSuffix: true
    });
};

export const filterPerson = (people: Person[], query: string): Person[] => {
    return people.filter((friend) => {
        if (friend.personaname.toLowerCase().includes(query)) {
            return true;
        } else if (friend.steamid.toString() == query) {
            return true;
        }
        // TODO convert steamids from other formats to query
        return false;
    });
};

export const steamIdQueryValue = (sid: string): string => {
    try {
        const s = new SteamID(sid);
        return `${s.getSteamID64()}-${s.getSteam2RenderedID()}-${s.getSteam3RenderedID()}`;
    } catch (_) {
        return '';
    }
};

export const stringHexNumber = (input: string) =>
    (
        '#' +
        (
            (('000000' as never) +
                parseInt(
                    // 2
                    parseInt(input, 36) // 3
                        .toExponential() // 4
                        .slice(2, -5), // 5
                    10
                )) &
            0xffffff
        ) // 6
            .toString(16)
            .toUpperCase()
    ).slice(-6); // "32EF01"     // 7

const humanize = (count: number, thresh: number, dp = 1, units: string[]) => {
    let u = -1;
    const r = 10 ** dp;

    // eslint-disable-next-line no-loops/no-loops
    do {
        count /= thresh;
        ++u;
    } while (
        Math.round(Math.abs(count) * r) / r >= thresh &&
        u < units.length - 1
    );

    return count.toFixed(dp) + '' + units[u];
};

export const humanFileSize = (bytes: number, si = false, dp = 1) => {
    const thresh = si ? 1000 : 1024;

    if (Math.abs(bytes) < thresh) {
        return bytes + ' B';
    }

    const units = si
        ? ['kB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB']
        : ['KiB', 'MiB', 'GiB', 'TiB', 'PiB', 'EiB', 'ZiB', 'YiB'];
    return humanize(bytes, thresh, dp, units);
};

export const humanCount = (count: number, dp: number = 1): string => {
    if (Math.abs(count) < 1000) {
        return `${count}`;
    }
    return humanize(count, 1000, dp, ['K', 'M', 'B', 'T', 'Q']);
};

export const defaultFloatFmtPct = (value: number) => `${value.toFixed(2)}%`;
export const defaultFloatFmt = (value: number) => value.toFixed(2);

export const isValidHttpURL = (value: string): boolean => {
    try {
        const url = new URL(value);
        return url.protocol === 'http:' || url.protocol === 'https:';
    } catch (_) {
        return false;
    }
};

export const steamIDOrEmptyString = async (steamId: string) => {
    if (emptyOrNullString(steamId)) {
        return '';
    }
    try {
        const resp = await apiGetProfile(steamId);
        const sid = new SteamID(resp.player.steam_id);
        return sid.getSteamID64();
    } catch (e) {
        return '';
    }
};

const ipRegex =
    /^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$/gi;

export const validIP = (value: string): boolean => ipRegex.test(value);
