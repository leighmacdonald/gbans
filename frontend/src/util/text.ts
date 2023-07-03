import format from 'date-fns/format';
import { formatDistance, parseISO, parseJSON } from 'date-fns';
import { Person } from '../api';
import SteamID from 'steamid';

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

export const humanFileSize = (bytes: number, si = false, dp = 1) => {
    const thresh = si ? 1000 : 1024;

    if (Math.abs(bytes) < thresh) {
        return bytes + ' B';
    }

    const units = si
        ? ['kB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB']
        : ['KiB', 'MiB', 'GiB', 'TiB', 'PiB', 'EiB', 'ZiB', 'YiB'];
    let u = -1;
    const r = 10 ** dp;

    // eslint-disable-next-line no-loops/no-loops
    do {
        bytes /= thresh;
        ++u;
    } while (
        Math.round(Math.abs(bytes) * r) / r >= thresh &&
        u < units.length - 1
    );

    return bytes.toFixed(dp) + ' ' + units[u];
};
