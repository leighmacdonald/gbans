import { Select } from '@mui/material';
import FormControl from '@mui/material/FormControl';
import FormHelperText from '@mui/material/FormHelperText';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import { useFormikContext } from 'formik';
import * as yup from 'yup';
import { AppealState, appealStateString } from '../../api';

export const appealStateFielValidator = yup
    .string()
    .label('Select a appeal state')
    .required('Appeal state is required');

export interface AppealStateFieldProps {
    appeal_state: AppealState;
}

export const AppealStateField = <T,>() => {
    const { values, touched, errors, handleChange } = useFormikContext<
        T & AppealStateFieldProps
    >();
    return (
        <FormControl fullWidth>
            <InputLabel id="appeal_state-label">Appeal Status</InputLabel>
            <Select<AppealState>
                fullWidth
                label={'Appeal Status'}
                labelId="appeal_state-label"
                id="appeal_state"
                value={values.appeal_state}
                name={'appeal_state'}
                onChange={handleChange}
                error={touched.appeal_state && Boolean(errors.appeal_state)}
            >
                {[
                    AppealState.Any,
                    AppealState.Open,
                    AppealState.Denied,
                    AppealState.Accepted,
                    AppealState.Reduced,
                    AppealState.NoAppeal
                ].map((state) => (
                    <MenuItem key={state} value={state}>
                        {appealStateString(state)}
                    </MenuItem>
                ))}
            </Select>
            <FormHelperText>
                {touched.appeal_state &&
                    errors.appeal_state &&
                    `${errors.appeal_state}`}
            </FormHelperText>
        </FormControl>
    );
};
