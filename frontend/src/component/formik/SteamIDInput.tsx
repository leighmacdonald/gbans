import React from 'react';
import TextField from '@mui/material/TextField';
import { FormikHandlers, FormikState } from 'formik/dist/types';
import { Nullable } from '../../util/types';
import { PlayerProfile } from '../../api';
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

export const SteamIDInput = ({
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
