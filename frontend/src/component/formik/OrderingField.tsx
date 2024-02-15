import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import * as yup from 'yup';

export const orderingFieldValidator = yup.number().label('Ordering').integer();

interface OrderingFieldProps {
    ordering: number;
}

export const OrderingField = <T,>() => {
    const { values, handleChange, touched, errors } = useFormikContext<
        T & OrderingFieldProps
    >();

    return (
        <TextField
            // disabled={values.ban_asn_id != undefined && values.ban_asn_id > 0}
            type={'number'}
            fullWidth
            label={'Ordering'}
            id={'ordering'}
            name={'ordering'}
            value={values.ordering}
            onChange={handleChange}
            error={touched.ordering && Boolean(errors.ordering)}
            helperText={
                touched.ordering &&
                Boolean(errors.ordering) &&
                `${errors.ordering}`
            }
        />
    );
};
