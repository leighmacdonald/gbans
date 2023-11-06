import React from 'react';
import { Select } from '@mui/material';
import FormControl from '@mui/material/FormControl';
import FormHelperText from '@mui/material/FormHelperText';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import { useFormikContext } from 'formik';
import * as yup from 'yup';
import { ReportStatus, reportStatusString } from '../../api';

export const reportStatusFielValidator = yup
    .string()
    .label('Select a report status')
    .required('report status is required');

export interface ReportStatusFieldProps {
    report_status: ReportStatus;
}

export const ReportStatusField = <T,>() => {
    const { values, touched, errors, handleChange } = useFormikContext<
        T & ReportStatusFieldProps
    >();
    return (
        <FormControl sx={{ width: 450 }}>
            <InputLabel id="report-status-label">Report status</InputLabel>
            <Select<ReportStatus>
                labelId="report-status-label"
                id="report_status"
                value={values.report_status}
                name={'report_status'}
                onChange={handleChange}
                error={touched.report_status && Boolean(errors.report_status)}
            >
                {[
                    ReportStatus.Any,
                    ReportStatus.Opened,
                    ReportStatus.NeedMoreInfo,
                    ReportStatus.ClosedWithoutAction,
                    ReportStatus.ClosedWithAction
                ].map((status) => (
                    <MenuItem key={status} value={status}>
                        {reportStatusString(status)}
                    </MenuItem>
                ))}
            </Select>
            <FormHelperText>
                {touched.report_status &&
                    errors.report_status &&
                    `${errors.report_status}`}
            </FormHelperText>
        </FormControl>
    );
};
