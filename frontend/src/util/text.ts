import format from "date-fns/format";
import fromUnixTime from "date-fns/fromUnixTime";

/**
 * Basic string formatting, similar to python format with dict
 * > fmt("test {a}", {"a": "bbb"})
 * < test bbb"
 *
 * @param s
 * @param args
 */
export function fmt(s: string, args: object): string {
    if (args) {
        for (let k in args) {
            s = s.replace(new RegExp("\\{" + k + "\\}", "gi"), args[k]);
        }
    }
    return s;
}

export function fmtUnix(t: number): string {
    return format(fromUnixTime(t), "MMM d y H:m")
}