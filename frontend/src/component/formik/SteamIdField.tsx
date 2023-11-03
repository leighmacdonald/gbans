import React from 'react';
import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import SteamID from 'steamid';
import * as yup from 'yup';
import { apiGetProfile, PlayerProfile } from '../../api';
import { logErr } from '../../util/errors';
import { Nullable } from '../../util/types';

export const steamIdValidator = yup
    .string()
    .test('checkSteamId', 'Invalid steamid/vanity', async (steamId, ctx) => {
        if (!steamId) {
            return false;
        }
        try {
            const resp = await apiGetProfile(steamId);
            const sid = new SteamID(resp.player.steam_id);
            ctx.parent.value = sid.getSteamID64();
            return true;
        } catch (e) {
            logErr(e);
            return false;
        }
    })
    .label('Enter your steam_id')
    .required('steam_id is required');

export interface BaseFormikInputProps {
    id?: string;
    label?: string;
    initialValue?: string;
    fullWidth: boolean;
    isReadOnly?: boolean;
    onProfileSuccess?: (profile: Nullable<PlayerProfile>) => void;
}
export interface SteamIDInputValue {
    steam_id: string;
}

export const SteamIdField = <T,>({ isReadOnly }: BaseFormikInputProps) => {
    const { values, touched, errors, handleChange } = useFormikContext<
        T & SteamIDInputValue
    >();
    return (
        <TextField
            fullWidth
            disabled={isReadOnly}
            name={'steam_id'}
            id={'steam_id'}
            label={'Steam ID / Profile'}
            value={values.steam_id}
            onChange={handleChange}
            error={touched.steam_id && Boolean(errors.steam_id)}
            //helperText={touched.steam_id && errors.steam_id}
        />
    );
};
