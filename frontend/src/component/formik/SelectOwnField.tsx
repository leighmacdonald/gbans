import FormControl from '@mui/material/FormControl';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormGroup from '@mui/material/FormGroup';
import FormHelperText from '@mui/material/FormHelperText';
import Switch from '@mui/material/Switch';
import { useFormikContext } from 'formik';
import { VCenterBox } from '../VCenterBox';

interface SelectOwnFieldProps {
    select_own: boolean;
    source_id?: string;
}

export const SelectOwnField = <T,>({ disabled }: { disabled?: boolean }) => {
    const { values, touched, errors, handleChange } = useFormikContext<
        T & SelectOwnFieldProps
    >();
    return (
        <VCenterBox>
            <FormControl fullWidth disabled={disabled}>
                <FormGroup>
                    <FormControlLabel
                        value={values.select_own}
                        id={'select_own'}
                        name={'select_own'}
                        onChange={handleChange}
                        control={<Switch checked={values.select_own} />}
                        label="Only Mine"
                    />
                </FormGroup>
                <FormHelperText>
                    {touched.select_own &&
                        errors.select_own &&
                        `${errors.select_own}`}
                </FormHelperText>
            </FormControl>
        </VCenterBox>
    );
};
