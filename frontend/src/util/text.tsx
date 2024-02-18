import { formatDistance, parseISO, parseJSON } from 'date-fns';
import format from 'date-fns/format';
import { isAfter } from 'date-fns/fp';
import { defaultAvatarHash, Person } from '../api';

export const parseDateTime = (t: string): Date => {
    return parseISO(t);
};

export const renderDateTime = (t: Date): string => {
    return format(t, 'yyyy-MM-dd HH:mm');
};

export const renderDate = (t: Date): string => {
    return format(t, 'yyyy-MM-dd');
};

export const renderTime = (t: Date): string => {
    return format(t, 'HH:mm');
};

export const isValidSteamDate = (date: Date) =>
    isAfter(new Date(2000, 0, 0), date);

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

type avatarSize = 'small' | 'medium' | 'full';

export const avatarHashToURL = (hash?: string, size: avatarSize = 'full') => {
    return `https://avatars.steamstatic.com/${hash ?? defaultAvatarHash}${
        size == 'small' ? '' : `_${size}`
    }.jpg`;
};
