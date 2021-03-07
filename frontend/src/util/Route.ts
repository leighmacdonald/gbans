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

export function vars(p: string, vars: any):string {
    if (!vars) {
        return p
    }
    let first = true
    for (let k in vars) {
        if (vars.hasOwnProperty(k)) {
            p += fmt("{p}{key}={value}", {
                p: first ? "?" : "&",
                key: k,
                value: vars[k]
            })
            first = false
        }
    }
    return p;
}