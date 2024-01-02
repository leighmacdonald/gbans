import React from 'react';
import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import * as yup from 'yup';
import { isValidIP } from '../../util/validators';

export const makeNetworkRangeFieldValidator = (required: boolean) => {
    return (
        required
            ? yup.string().required('CIDR address is required')
            : yup.string().optional()
    )
        .label('Input a CIDR network range')
        .test('rangeValid', 'IP / CIDR invalid', (addr) => {
            if (addr == undefined && !required) {
                return true;
            }
            if (!addr) {
                return false;
            }
            if (!addr.includes('/')) {
                addr = addr + '/32';
            }

            const v = addr.split('/');
            if (!isValidIP(v[0])) {
                return false;
            }
            return !(v.length > 1 && parseInt(v[1]) < 24);
        });
};

export interface CIDRInputFieldProps {
    cidr: string;
}

export const NetworkRangeField = <T,>() => {
    const { values, touched, errors, handleChange } = useFormikContext<
        T & CIDRInputFieldProps
    >();
    return (
        <TextField
            fullWidth
            label={'CIDR Network Range'}
            id={'cidr'}
            name={'cidr'}
            value={values.cidr}
            onChange={handleChange}
            error={touched.cidr && Boolean(errors.cidr)}
            helperText={touched.cidr && errors.cidr && `${errors.cidr}`}
        />
    );
};
