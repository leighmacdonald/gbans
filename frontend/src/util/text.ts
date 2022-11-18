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

export const steamIdQueryValue = (sid: SteamID): string =>
    `${sid.getSteamID64()}-${sid.getSteam2RenderedID()}-${sid.getSteam3RenderedID()}`;

export const stringHexNumber = (input: string) =>
    (
        '#' +
        (
            (('000000' as any) +
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
