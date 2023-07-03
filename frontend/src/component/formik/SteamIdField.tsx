import React from 'react';
import TextField from '@mui/material/TextField';
import { FormikHandlers, FormikState } from 'formik/dist/types';
import { Nullable } from '../../util/types';
import { apiGetProfile, PlayerProfile } from '../../api';
import * as yup from 'yup';
import { logErr } from '../../util/errors';
import SteamID from 'steamid';

export const steamIdValidator = yup
    .string()
    .test('checkSteamId', 'Invalid steamid/vanity', async (steamId, ctx) => {
        if (!steamId) {
            return false;
        }
        try {
            const resp = await apiGetProfile(steamId);
            if (!resp.status || !resp.result) {
                return false;
            }
            const sid = new SteamID(resp.result.player.steam_id);
            ctx.parent.value = sid.getSteamID64();
            return true;
        } catch (e) {
            logErr(e);
            return false;
        }
    })
    .label('Enter your steam_id')
    .required('steam_id is required');

export interface SteamIDInputProps<T> {
    id?: string;
    label?: string;
    initialValue?: string;
    fullWidth: boolean;
    isReadOnly?: boolean;
    onProfileSuccess?: (profile: Nullable<PlayerProfile>) => void;
    formik: FormikState<T> & FormikHandlers;
}

export interface SteamIDInputValue {
    steam_id: string;
}

export const SteamIdField = ({
    id,
    formik,
    isReadOnly
}: SteamIDInputProps<SteamIDInputValue>) => {
    return (
        <TextField
            fullWidth
            disabled={isReadOnly ?? false}
            name={id ?? 'steam_id'}
            id={id ?? 'steam_id'}
            label={'Steam ID / Profile'}
            value={formik.values.steam_id}
            onChange={formik.handleChange}
            error={formik.touched.steam_id && Boolean(formik.errors.steam_id)}
            helperText={formik.touched.steam_id && formik.errors.steam_id}
        />
    );
};
