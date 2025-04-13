import { formatDistance, parseISO, parseJSON } from 'date-fns';
import { format } from 'date-fns/format';
import { isAfter } from 'date-fns/fp';

export type Duration = number;

export const durationToMs = (d: Duration) => d / 1000;

export const durationToSec = (d: Duration) => d / 1000 / 1000;

export const durationString = (d: Duration) => {
    const secs = durationToSec(d);
    let hours: number | string = Math.floor(secs / 3600);
    let minutes: number | string = Math.floor((secs - hours * 3600) / 60);
    let seconds: number | string = secs - hours * 3600 - minutes * 60;

    if (hours < 10) {
        hours = '0' + hours;
    }

    if (minutes < 10) {
        minutes = '0' + minutes;
    }

    if (seconds < 10) {
        seconds = '0' + seconds;
    }

    return hours + ':' + minutes + ':' + seconds;
};

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

export const isValidSteamDate = (date: Date) => isAfter(new Date(2000, 0, 0), date);

export const renderTimeDistance = (t1: Date | string, t2?: Date | string): string => {
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

export interface DateRange {
    date_start: Date;
    date_end: Date;
}

export const transformDateRange = <T>(item: T & DateRange) => {
    item.date_end = parseDateTime(item.date_end as unknown as string);
    item.date_start = parseDateTime(item.date_start as unknown as string);

    return item;
};

export interface TimeStamped {
    created_on: Date;
    updated_on: Date;
    valid_until?: Date;
}

export const transformCreatedOnDate = <T>(item: T & { created_on: Date }) => {
    item.created_on = parseDateTime(item.created_on as unknown as string);
    return item;
};

export const transformCreatedAtDate = <T>(item: T & { created_at: Date }) => {
    item.created_at = parseDateTime(item.created_at as unknown as string);
    return item;
};

export const transformTimeStampedDates = <T>(item: T & TimeStamped) => {
    item.created_on = parseDateTime(item.created_on as unknown as string);
    item.updated_on = parseDateTime(item.updated_on as unknown as string);
    if (item.valid_until != undefined) {
        item.valid_until = parseDateTime(item.valid_until as unknown as string);
    }
    return item;
};

export const transformTimeStampedDatesList = <T>(items: (T & TimeStamped)[]) => {
    return items ? items.map(transformTimeStampedDates) : items;
};
