import Grid from '@mui/material/Grid';
import Stack from '@mui/material/Stack';
import React, { useState } from 'react';
import Typography from '@mui/material/Typography';
import Box from '@mui/material/Box';
import { Person, Report, ReportWithAuthor } from '../api';

interface AdminReportRowProps {
    report: Report;
    author: Person;
}

const AdminReportRow = ({ report }: AdminReportRowProps): JSX.Element => {
    return (
        <Box>
            <Typography padding={2} variant={'h5'}>
                {report.title}
            </Typography>
        </Box>
    );
};

export const AdminReports = (): JSX.Element => {
    const [reports] = useState<ReportWithAuthor[]>([]);
    return (
        <Grid container>
            <Grid item xs={8}>
                <Stack spacing={2}>
                    {reports.map((r) => {
                        return (
                            <AdminReportRow
                                key={r.report.report_id}
                                author={r.author}
                                report={r.report}
                            />
                        );
                    })}
                </Stack>
            </Grid>
        </Grid>
    );
};
