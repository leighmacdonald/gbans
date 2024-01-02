import SteamID from 'steamid';
import { apiGetProfile } from '../api';
import { logErr } from './errors';
import { emptyOrNullString } from './types';

export const isValidHttpURL = (value: string): boolean => {
    try {
        const url = new URL(value);
        return url.protocol === 'http:' || url.protocol === 'https:';
    } catch (_) {
        return false;
    }
};

const profileIDRx = new RegExp(
    /https:\/\/steamcommunity.com\/profiles\/(\d+)\/?/
);

const profileVanityRx = new RegExp(
    /https:\/\/steamcommunity.com\/id\/(\w+)\/?/
);

const hasWhiteSpace = (s: string) => /\s/.test(s);

/**
 * Parse and validate any steamid input type, supports all steamid formats including:
 *
 * - https://steamcommunity.com/profiles/76561197960287930
 * - https://steamcommunity.com/id/gabelogannewell
 * - Bare vanity name: gabelogannewell
 * - SteamID64: 76561197960287930
 * - SteamID32: [U:1:22202]
 * - SteamID: STEAM_0:0:11101
 *
 * @param input
 * @param individualOnly Only consider ids belonging to individuals in public universe as valid
 */
export const steamIDOrEmptyString = async (
    input: string,
    individualOnly: boolean = true
) => {
    const steamIdInput = input.trimEnd();

    if (emptyOrNullString(steamIdInput)) {
        return '';
    } else if (hasWhiteSpace(steamIdInput)) {
        return '';
    }

    let steamId = '';

    // Initial basic check for bare steam ids
    try {
        const sid = new SteamID(steamId);
        if (individualOnly ? sid.isValidIndividual() : sid.isValid()) {
            return sid.getSteamID64();
        }
    } catch (_) {
        // ignored
    }

    // Check for vanity url and test it against api
    const vanity = profileVanityRx.exec(steamIdInput);
    if (vanity != null) {
        try {
            const resp = await apiGetProfile(vanity[1]);
            steamId = resp.player.steam_id;
        } catch (e) {
            logErr(e);
            return '';
        }
    }

    // Check for steamid in profile url
    if (steamId == '') {
        const matches = profileIDRx.exec(steamIdInput);
        if (matches != null) {
            steamId = matches[1];
        }
    }

    // Attempt to verify as a vanity name if its bare string
    if (steamId == '') {
        try {
            const resp = await apiGetProfile(steamIdInput);
            steamId = resp.player.steam_id;
        } catch (e) {
            logErr(e);
            return '';
        }
    }

    // Validate any parsed steamid value
    if (steamId != '') {
        try {
            const sid = new SteamID(steamId);
            if (individualOnly ? sid.isValidIndividual() : sid.isValid()) {
                return sid.getSteamID64();
            }
        } catch (e) {
            logErr(e);
            return '';
        }
    }

    return '';
};

const ipRegex =
    /^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$/gi;

export const isValidIP = (value: string): boolean => ipRegex.test(value);