import FormControl from '@mui/material/FormControl';
import FormHelperText from '@mui/material/FormHelperText';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import { useFormikContext } from 'formik';
import * as yup from 'yup';
import { BanType } from '../../api';

export const BanTypeFieldValidator = yup.number().label('Select a ban type').required('ban type is required');

interface BanTypeFieldProps {
    ban_type: BanType;
}

export const BanTypeField = <T,>() => {
    const { values, touched, errors, handleChange } = useFormikContext<T & BanTypeFieldProps>();

    return (
        <FormControl fullWidth>
            <InputLabel id="actionType-label">Action Type</InputLabel>
            <Select<BanType>
                fullWidth
                label={'Action Type'}
                labelId="actionType-label"
                id="ban_type"
                name={'ban_type'}
                value={values.ban_type}
                onChange={handleChange}
                error={touched.ban_type && Boolean(errors.ban_type)}
                defaultValue={BanType.Banned}
            >
                {[BanType.Banned, BanType.NoComm].map((v) => (
                    <MenuItem key={`time-${v}`} value={v}>
                        {v == BanType.NoComm ? 'Mute' : 'Ban'}
                    </MenuItem>
                ))}
            </Select>
            <FormHelperText>{touched.ban_type && errors.ban_type && `${errors.ban_type}`}</FormHelperText>
        </FormControl>
    );
};
