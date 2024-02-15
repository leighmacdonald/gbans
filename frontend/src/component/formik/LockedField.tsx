import FormControl from '@mui/material/FormControl';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormGroup from '@mui/material/FormGroup';
import FormHelperText from '@mui/material/FormHelperText';
import Switch from '@mui/material/Switch';
import { useFormikContext } from 'formik';
import { VCenterBox } from '../VCenterBox';

interface LockedFieldProps {
    locked: boolean;
}

export const LockedField = <T,>() => {
    const { values, touched, errors, handleChange } = useFormikContext<
        T & LockedFieldProps
    >();
    return (
        <VCenterBox>
            <FormControl fullWidth>
                <FormGroup>
                    <FormControlLabel
                        value={values.locked}
                        id={'locked'}
                        name={'locked'}
                        onChange={handleChange}
                        control={<Switch checked={values.locked} />}
                        label="Locked Thread"
                    />
                </FormGroup>
                <FormHelperText>
                    {touched.locked && errors.locked && `${errors.locked}`}
                </FormHelperText>
            </FormControl>
        </VCenterBox>
    );
};
