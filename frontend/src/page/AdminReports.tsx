import Grid from '@mui/material/Grid';
import Stack from '@mui/material/Stack';
import React, { useEffect, useMemo, useState } from 'react';
import Typography from '@mui/material/Typography';
import {
    apiGetReports,
    ReportStatus,
    reportStatusString,
    ReportWithAuthor
} from '../api';
import { logErr } from '../util/errors';
import { UserTable } from '../component/UserTable';
import Paper from '@mui/material/Paper';
import format from 'date-fns/format';
import { parseISO } from 'date-fns';
import { useNavigate } from 'react-router-dom';
import { Heading } from '../component/Heading';
import { Select } from '@mui/material';
import MenuItem from '@mui/material/MenuItem';
import { SelectChangeEvent } from '@mui/material/Select';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';

export const AdminReports = (): JSX.Element => {
    const [reports, setReports] = useState<ReportWithAuthor[]>([]);
    const [filterStatus, setFilterStatus] = useState(ReportStatus.Any);
    const navigate = useNavigate();

    useEffect(() => {
        apiGetReports({})
            .then((reports) => {
                setReports(reports || []);
            })
            .catch(logErr);
    }, []);

    const cachedReports = useMemo(() => {
        if (filterStatus == ReportStatus.Any) {
            return reports;
        }
        return reports.filter((r) => r.report.report_status == filterStatus);
    }, [filterStatus, reports]);

    const handleReportStateChange = (
        event: SelectChangeEvent<ReportStatus>
    ) => {
        setFilterStatus(event.target.value as ReportStatus);
    };

    return (
        <Grid container spacing={3} paddingTop={3}>
            <Grid item xs={12}>
                <Stack spacing={2}>
                    <Paper>
                        <Heading>Current User Reports</Heading>
                        <FormControl sx={{ padding: 2 }}>
                            <InputLabel id="report-status-label">
                                Report status
                            </InputLabel>
                            <Select<ReportStatus>
                                labelId="report-status-label"
                                id="report-status-select"
                                value={filterStatus}
                                onChange={handleReportStateChange}
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
                        </FormControl>

                        <UserTable
                            onRowClick={(report) => {
                                navigate(`/report/${report.report.report_id}`);
                            }}
                            columns={[
                                {
                                    label: 'Status',
                                    tooltip: 'Status',
                                    sortType: 'string',
                                    align: 'left',
                                    width: '200px',
                                    renderer: (obj) => {
                                        return (
                                            <Typography variant={'subtitle1'}>
                                                {reportStatusString(
                                                    obj.report.report_status
                                                )}
                                            </Typography>
                                        );
                                    }
                                },
                                {
                                    label: 'Created',
                                    tooltip: 'Created On',
                                    sortType: 'date',
                                    align: 'left',
                                    width: '150px',
                                    virtual: true,
                                    virtualKey: 'created_on',
                                    renderer: (obj) => {
                                        return (
                                            <Typography variant={'body1'}>
                                                {format(
                                                    parseISO(
                                                        obj.report
                                                            .created_on as any as string
                                                    ),
                                                    "yyyy-MM-dd'T'HH:mm"
                                                )}
                                            </Typography>
                                        );
                                    }
                                },
                                {
                                    label: 'Subject',
                                    tooltip: 'Subject',
                                    sortType: 'string',
                                    align: 'left',
                                    width: '250px',
                                    renderer: (obj) => {
                                        return (
                                            <Typography variant={'body1'}>
                                                <img src={obj.subject.avatar} />
                                                {obj.subject.personaname}
                                            </Typography>
                                        );
                                    }
                                },
                                {
                                    label: 'Reporter',
                                    tooltip: 'Reporter',
                                    sortType: 'string',
                                    align: 'left',
                                    renderer: (obj) => {
                                        return (
                                            <Typography variant={'body1'}>
                                                <img src={obj.author.avatar} />
                                                {obj.author.personaname}
                                            </Typography>
                                        );
                                    }
                                }
                            ]}
                            defaultSortColumn={'report'}
                            rowsPerPage={100}
                            rows={cachedReports}
                        />
                    </Paper>
                </Stack>
            </Grid>
        </Grid>
    );
};
