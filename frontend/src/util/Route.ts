import {fmt} from "./text";

export const Route = {
    Home: "/",
    Dist: "/dist",
    APIAppeal: "/api/appeal",
    APIBans: "/api/bans",
    APIFilteredWords: "/api/filtered_words"
}

export function route(r: any, args: null|object): string {
    return fmt(r, args);
}

export function vars(p: string, args: Record<string, any>):string {
    if (!args) {
        return p
    }
    const rv = args
        .filter((k) => args.hasOwnProperty(k))
        .map((k, v) => fmt("{k}={v}", {k, v}))
        .join("&")
    return rv ? "?" + rv : p;
}