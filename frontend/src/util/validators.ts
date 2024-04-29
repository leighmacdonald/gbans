import SteamID, { Type } from 'steamid';
import * as yup from 'yup';
import { apiGetProfile, BanReason } from '../api';
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

export const banReasonFieldValidator = yup.string().label('Select a reason').required('reason is required');

export const appealStateFielValidator = yup.string().label('Select a appeal state').required('Appeal state is required');

export const ipFieldValidator = yup.string().test('valid_ip', 'Invalid IP', (value) => {
    if (emptyOrNullString(value)) {
        return true;
    }
    return isValidIP(value as string);
});

export const unbanReasonTextFieldValidator = yup.string().min(5, 'Message to short').label('Unban Reason').required('Reason is required');

export const unbanValidationSchema = yup.object({
    unban_reason: unbanReasonTextFieldValidator
});

export const banReasonTextFieldValidator = yup.string().test('reason_text', '${path} invalid', (value, context) => {
    if (context.parent.reason != BanReason.Custom) {
        return true;
    } else {
        return value != undefined && value.length > 3;
    }
});

export const discordRequiredValidator = yup.boolean().label('Is discord required').required();

export const weightFieldValidator = yup.number().min(1, 'Min weight is 1').required('Weight required').label('Weight');

export const serverIDsValidator = yup.array().label('Select a server');

export const mapNameFieldValidator = yup.string().label('Select a map').min(3, 'Minimum 3 characters required').optional();

export const titleFieldValidator = yup.string().min(3, 'Title to short').label('Title').required('Title is required');

export const mapValidator = yup
    .string()
    .test('checkMap', 'Invalid map selection', async (map) => {
        return !emptyOrNullString(map);
    })
    .label('Select a map to play')
    .required('map is required');

export const makeNetworkRangeFieldValidator = (required: boolean) => {
    return (required ? yup.string().required('CIDR address is required') : yup.string().optional())
        .label('Input a CIDR network range')
        .test('rangeValid', 'IP / CIDR invalid', (addr) => {
            if (addr == undefined && !required) {
                return true;
            }
            if (!addr) {
                return false;
            }
            if (!addr.includes('/')) {
                addr = addr + '/32';
            }

            const v = addr.split('/');
            if (!isValidIP(v[0])) {
                return false;
            }
            return !(v.length > 1 && parseInt(v[1]) < 24);
        });
};

export const orderingFieldValidator = yup.number().label('Ordering').integer();

export const personanameFieldValidator = yup.string().min(3, 'Minimum length 3').label('Name Query');

export const reportStatusFieldValidator = yup.string().label('Select a report status').required('report status is required');

export const selectOwnValidator = yup.boolean().label('Include only results with yourself').required();

export const steamIdValidator = (attr_name: string = 'steam_id') => {
    return yup
        .string()
        .test('checkSteamId', 'Invalid steamid or profile url', async (steamId, ctx) => {
            if (!steamId) {
                return false;
            }
            try {
                const sid = await steamIDOrEmptyString(steamId);
                if (sid == '') {
                    return false;
                }
                ctx.parent[attr_name] = sid;
                return true;
            } catch (e) {
                logErr(e);
                return false;
            }
        })
        .label('Enter your Steam ID')
        .required('Steam ID is required');
};

export const filterActionValidator = yup.number().label('Select a action').required('Filter action is required');

export const deletedValidator = yup.boolean().label('Include deleted results').required();

export const asNumberFieldValidator = yup
    .number()
    .label('AS Number')
    .test('valid_asn', (value, ctx) => {
        if (value == undefined) {
            return true;
        }
        if (value <= 0) {
            return ctx.createError({ message: 'Invalid ASN' });
        }
        return true;
    })
    .integer();

export const groupIdFieldValidator = yup
    .string()
    .test('valid_group', 'Invalid group ID', (value) => {
        if (emptyOrNullString(value)) {
            return true;
        }
        try {
            const id = new SteamID(value as string);
            return id.isValid() && id.type == Type.CLAN;
        } catch (e) {
            logErr(e);
            return false;
        }
    })
    .length(18, 'Must be positive integer with a length of 18');

export const nonResolvingSteamIDInputTest = async (steamId: string | undefined) => {
    // Only validate once there is data.
    if (emptyOrNullString(steamId)) {
        return true;
    }
    try {
        const sid = new SteamID(steamId as string);
        return sid.isValidIndividual();
    } catch (e) {
        return false;
    }
};

export const targetIdValidator = yup
    .string()
    .label('Target Steam ID')
    .test('checkTargetId', 'Invalid target steamid', nonResolvingSteamIDInputTest);

export const sourceIdValidator = yup
    .string()
    .label('Author Steam ID')
    .test('source_id', 'Invalid author steamid', nonResolvingSteamIDInputTest);

export const steamIDValidatorSimple = yup
    .string()
    .label('Player Steam ID')
    .test('steam_id', 'Invalid steamid', nonResolvingSteamIDInputTest);

export const bodyMDValidator = yup.string().min(3, 'Message to short');

const profileIDRx = new RegExp(/https:\/\/steamcommunity.com\/profiles\/(\d+)\/?/);

const profileVanityRx = new RegExp(/https:\/\/steamcommunity.com\/id\/(\w+)\/?/);

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
export const steamIDOrEmptyString = async (input: string, individualOnly: boolean = true) => {
    const steamIdInput = input.trim();

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

    return steamId;
};

const ipRegex = /^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$/gi;

export const isValidIP = (value: string): boolean => ipRegex.test(value);
