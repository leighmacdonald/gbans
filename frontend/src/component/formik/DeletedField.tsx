import FormControl from '@mui/material/FormControl';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormGroup from '@mui/material/FormGroup';
import FormHelperText from '@mui/material/FormHelperText';
import Switch from '@mui/material/Switch';
import { useFormikContext } from 'formik';
import * as yup from 'yup';
import { VCenterBox } from '../VCenterBox';

export const deletedValidator = yup
    .boolean()
    .label('Include deleted results')
    .required();

interface DeletedFieldProps {
    deleted: boolean;
}

export const DeletedField = <T,>() => {
    const { values, touched, errors, handleChange } = useFormikContext<
        T & DeletedFieldProps
    >();
    return (
        <VCenterBox>
            <FormControl fullWidth>
                <FormGroup>
                    <FormControlLabel
                        value={values.deleted}
                        id={'deleted'}
                        name={'deleted'}
                        onChange={handleChange}
                        control={<Switch checked={values.deleted} />}
                        label="Include Deleted"
                    />
                </FormGroup>
                <FormHelperText>
                    {touched.deleted && errors.deleted && `${errors.deleted}`}
                </FormHelperText>
            </FormControl>
        </VCenterBox>
    );
};
