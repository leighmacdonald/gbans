import format from 'date-fns/format';
import { formatDistance, parseISO, parseJSON } from 'date-fns';
import { Person } from '../api';

export const parseDateTime = (t: string): Date => {
    return parseISO(t);
};

export const renderTime = (t: Date): string => {
    return format(t, 'Y-M-d HH:mm');
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
