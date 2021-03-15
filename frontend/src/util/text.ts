import format from 'date-fns/format';
import { formatDistance, parseISO } from 'date-fns';

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
        t1 = parseDateTime(t1);
    }
    if (!t2) {
        t2 = new Date();
    }
    if (typeof t2 === 'string') {
        t2 = parseDateTime(t2);
    }
    return formatDistance(t1, t2);
};
