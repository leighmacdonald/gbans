import format from "date-fns/format";
import {formatDistance, parseISO} from "date-fns";

/**
 * Basic string formatting, similar to python format with dict
 * > fmt("test {a}", {"a": "bbb"})
 * < test bbb"
 *
 * @param s
 * @param args
 */
export function fmt(s: string, args: any): string {
    if (args) {
        for (const k in args) {
            if (args.hasOwnProperty(k)) {
                s = s.replace(new RegExp("\\{" + k + "\\}", "gi"), args[k]);
            }
        }
    }
    return s;
}


export const parseDateTime = (t: string): Date => {
    return parseISO(t);
}

export const renderTime = (t: Date): string => {
    return format(t, "Y-M-d HH:mm");
}

export const renderTimeDistance = (t1: Date | string, t2?: Date | string): string => {
    if (typeof t1 === "string") {
        t1 = parseDateTime(t1);
    }
    if (!t2) {
        t2 = new Date();
    }
    if (typeof t2 === "string") {
        t2 = parseDateTime(t2);
    }
    return formatDistance(t1, t2)
}