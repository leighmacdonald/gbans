import React from 'react';
import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import * as yup from 'yup';
import { PlayerProfile } from '../../api';
import { logErr } from '../../util/errors';
import { Nullable } from '../../util/types';
import { steamIDOrEmptyString } from '../../util/validators';

export const steamIdValidator = yup
    .string()
    .test(
        'checkSteamId',
        'Invalid steamid or profile url',
        async (steamId, ctx) => {
            if (!steamId) {
                return false;
            }
            try {
                const sid = await steamIDOrEmptyString(steamId);
                if (sid == '') {
                    return false;
                }
                ctx.parent.value = sid;
                return true;
            } catch (e) {
                logErr(e);
                return false;
            }
        }
    )
    .label('Enter your Steam ID')
    .required('Steam ID is required');

export interface BaseFormikInputProps {
    id?: string;
    label?: string;
    initialValue?: string;
    isReadOnly?: boolean;
    onProfileSuccess?: (profile: Nullable<PlayerProfile>) => void;
}
export interface SteamIDInputValue {
    steam_id: string;
}

export const SteamIdField = ({ isReadOnly = false }: BaseFormikInputProps) => {
    const { values, touched, errors, handleChange } =
        useFormikContext<SteamIDInputValue>();
    return (
        <TextField
            fullWidth
            disabled={isReadOnly}
            name={'steam_id'}
            id={'steam_id'}
            label={'Steam ID / Profile URL / Vanity Name'}
            value={values.steam_id}
            onChange={handleChange}
            error={touched.steam_id && Boolean(errors.steam_id)}
            helperText={
                touched.steam_id && errors.steam_id && `${errors.steam_id}`
            }
        />
    );
};
