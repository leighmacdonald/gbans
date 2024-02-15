import FormControl from '@mui/material/FormControl';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormGroup from '@mui/material/FormGroup';
import FormHelperText from '@mui/material/FormHelperText';
import Switch from '@mui/material/Switch';
import { useFormikContext } from 'formik';
import { VCenterBox } from '../VCenterBox';

interface StickyFieldProps {
    sticky: boolean;
}

export const StickyField = <T,>() => {
    const { values, touched, errors, handleChange } = useFormikContext<
        T & StickyFieldProps
    >();
    return (
        <VCenterBox>
            <FormControl fullWidth>
                <FormGroup>
                    <FormControlLabel
                        value={values.sticky}
                        id={'sticky'}
                        name={'sticky'}
                        onChange={handleChange}
                        control={<Switch checked={values.sticky} />}
                        label="Sticky Thread"
                    />
                </FormGroup>
                <FormHelperText>
                    {touched.sticky && errors.sticky && `${errors.sticky}`}
                </FormHelperText>
            </FormControl>
        </VCenterBox>
    );
};
