import {fmt} from "./text";

export const Route = {
    Home: "/",
    Dist: "/dist",
    APIBans: "/api/v1/bans",
    APIFilteredWords: "/api/v1/filtered_words"
}

export function route(r: any, args: null|object): string {
    return fmt(r, args);
}

export function vars(p: string, vars: object):string {
    if (!vars) {
        return p
    }
    let first = true
    for (let k in vars) {
        p += fmt("{p}{key}={value}", {
            p: first ? "?" : "&",
            key: k,
            value: vars[k]
        })
        first = false
    }
    return p;
}