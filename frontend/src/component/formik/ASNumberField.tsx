import React from 'react';
import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import * as yup from 'yup';

export const asNumberFieldValidator = yup
    .number()
    .label('AS Number')
    .test('valid_asn', (value, ctx) => {
        if (value == undefined) {
            return true;
        }
        if (value <= 0) {
            return ctx.createError({ message: 'Invalid ASN' });
        }
        return true;
    })
    .integer();

interface ASNumberFieldProps {
    as_num: number;
    ban_asn_id?: number;
}

export const ASNumberField = <T,>() => {
    const { values, handleChange, touched, errors } = useFormikContext<
        T & ASNumberFieldProps
    >();
    return (
        <TextField
            // disabled={values.ban_asn_id != undefined && values.ban_asn_id > 0}
            type={'number'}
            fullWidth
            label={'Autonomous System Number'}
            id={'as_num'}
            name={'as_num'}
            value={values.as_num}
            onChange={handleChange}
            error={touched.as_num && Boolean(errors.as_num)}
            helperText={
                touched.as_num && Boolean(errors.as_num) && `${errors.as_num}`
            }
        />
    );
};
