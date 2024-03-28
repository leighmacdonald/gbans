import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';

export type TargetIDInputValue = {
    target_id: string;
};

export const TargetIDField = ({
    label = 'Target Steam ID',
    isReadOnly = false
}: {
    label?: string;
    isReadOnly?: boolean;
}) => {
    const { values, touched, errors, handleChange } =
        useFormikContext<TargetIDInputValue>();
    return (
        <TextField
            fullWidth
            name={'target_id'}
            id={'target_id'}
            label={label}
            disabled={isReadOnly}
            value={values.target_id}
            onChange={handleChange}
            error={touched.target_id && Boolean(errors.target_id)}
            helperText={
                touched.target_id && errors.target_id && `${errors.target_id}`
            }
        />
    );
};
