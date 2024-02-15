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

export const sourceIdValidator = yup
    .string()
    .label('Author Steam ID')
    .test('source_id', 'Invalid author steamid', nonResolvingSteamIDInputTest);

interface AuthorIDFieldValue {
    source_id: string;
}

export const SourceIdField = <T,>({
    disabled = false
}: {
    disabled?: boolean;
}) => {
    const { values, touched, errors, handleChange } = useFormikContext<
        T & AuthorIDFieldValue
    >();
    return (
        <TextField
            variant={'outlined'}
            fullWidth
            disabled={disabled}
            name={'source_id'}
            id={'source_id'}
            label={'Author Steam ID'}
            value={values.source_id}
            onChange={handleChange}
            error={touched.source_id && Boolean(errors.source_id)}
            helperText={
                touched.source_id && errors.source_id && `${errors.source_id}`
            }
        />
    );
};
