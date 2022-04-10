import Grid from '@mui/material/Grid';
import Stack from '@mui/material/Stack';
import React, { useEffect, useState } from 'react';
import Typography from '@mui/material/Typography';
import {
    apiGetReports,
    Person,
    renderDate,
    Report,
    ReportQueryFilter,
    ReportStatus,
    ReportWithAuthor,
    Team
} from '../api';
import Paper from '@mui/material/Paper';
import { ProfileButton } from '../component/ProfileButton';
import ButtonGroup from '@mui/material/ButtonGroup';
import PageviewIcon from '@mui/icons-material/Pageview';
import IconButton from '@mui/material/IconButton';
import { useNavigate } from 'react-router-dom';
import { noop } from 'lodash-es';
interface AdminReportRowProps {
    report: Report;
    author: Person;
}

const AdminReportRow = ({
    report,
    author
}: AdminReportRowProps): JSX.Element => {
    const navigate = useNavigate();
    return (
        <Paper elevation={1}>
            <Stack direction={'row'}>
                <ButtonGroup>
                    <IconButton
                        focusRipple={false}
                        color={'primary'}
                        onClick={() => {
                            navigate(`/report/${report.report_id}`);
                        }}
                    >
                        <PageviewIcon />
                    </IconButton>
                </ButtonGroup>
                <Typography padding={2} variant={'h5'}>
                    {renderDate(report.created_on)}
                </Typography>
                <Typography padding={2} variant={'h5'}>
                    {report.title}
                </Typography>
                <ProfileButton
                    source={author}
                    team={Team.SPEC}
                    setFilter={noop}
                />
            </Stack>
        </Paper>
    );
};

export const AdminReports = (): JSX.Element => {
    const [reports, setReports] = useState<ReportWithAuthor[]>([]);
    const [reportStatus] = useState<ReportStatus>(ReportStatus.Opened);

    useEffect(() => {
        const f = async () => {
            const opts: ReportQueryFilter = { report_status: reportStatus };
            const reports = await apiGetReports(opts);
            setReports(reports);
        };
        f();
    });

    return (
        <Grid container>
            <Grid item xs={8}>
                <Stack spacing={2}>
                    <Typography variant={'h3'}>Current User Reports</Typography>
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
