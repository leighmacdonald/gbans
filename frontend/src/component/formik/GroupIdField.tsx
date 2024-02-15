import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import SteamID, { Type } from 'steamid';
import * as yup from 'yup';
import { logErr } from '../../util/errors';
import { emptyOrNullString } from '../../util/types';

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

interface GroupIDFieldProps {
    group_id: string;
    ban_group_id?: number;
}

export const GroupIdField = <T,>() => {
    const { values, touched, errors, handleChange } = useFormikContext<
        T & GroupIDFieldProps
    >();
    return (
        <TextField
            fullWidth
            disabled={
                values.ban_group_id != undefined && values.ban_group_id > 0
            }
            id="group_id"
            name={'group_id'}
            label="Steam Group ID"
            value={values.group_id}
            onChange={handleChange}
            error={touched.group_id && Boolean(errors.group_id)}
            helperText={
                touched.group_id && errors.group_id && `${errors.group_id}`
            }
            variant="outlined"
        />
    );
};
