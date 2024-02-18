import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import { PlayerProfile } from '../../api';
import { Nullable } from '../../util/types';

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
