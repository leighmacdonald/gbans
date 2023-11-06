import React from 'react';
import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import SteamID from 'steamid';
import * as yup from 'yup';
import { emptyOrNullString } from '../../util/types';

export const nonResolvingSteamIDInputTest = async (
    steamId: string | undefined
) => {
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

export const authorIdValidator = yup
    .string()
    .label('Author Steam ID')
    .test('author_id', 'Invalid author steamid', nonResolvingSteamIDInputTest);

interface AuthorIDFieldValue {
    author_id: string;
}

export const AuthorIDField = <T,>() => {
    const { values, touched, errors, handleChange } = useFormikContext<
        T & AuthorIDFieldValue
    >();
    return (
        <TextField
            variant={'outlined'}
            fullWidth
            name={'author_id'}
            id={'author_id'}
            label={'Author Steam ID'}
            value={values.author_id}
            onChange={handleChange}
            error={touched.author_id && Boolean(errors.author_id)}
            helperText={
                touched.author_id && errors.author_id && `${errors.author_id}`
            }
        />
    );
};
