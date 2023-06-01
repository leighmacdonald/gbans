import React, { useEffect, useMemo, useState } from 'react';
import Grid from '@mui/material/Grid';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import {
    apiGetReports,
    ReportStatus,
    reportStatusString,
    ReportWithAuthor
} from '../api';
import { logErr } from '../util/errors';
import { DataTable } from '../component/DataTable';
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
import Link from '@mui/material/Link';
import { PersonCell } from '../component/PersonCell';

export const AdminReports = () => {
    const [reports, setReports] = useState<ReportWithAuthor[]>([]);
    const [filterStatus, setFilterStatus] = useState(ReportStatus.Opened);
    const navigate = useNavigate();

    useEffect(() => {
        apiGetReports({})
            .then((response) => {
                setReports(response.result || []);
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
        <Grid container spacing={2} paddingTop={3}>
            <Grid item xs={12}>
                <Stack spacing={2}>
                    <Stack direction={'row'}>
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
                    </Stack>
                    <Paper>
                        <Heading>Current User Reports</Heading>

                        <DataTable
                            columns={[
                                {
                                    label: 'ID',
                                    tooltip: 'Report ID',
                                    sortType: 'number',
                                    align: 'left',
                                    queryValue: (o) => `${o.report.report_id}`,
                                    renderer: (obj) => (
                                        <Typography
                                            variant={'subtitle1'}
                                            component={Link}
                                            onClick={() => {
                                                navigate(
                                                    `/report/${obj.report.report_id}`
                                                );
                                            }}
                                        >
                                            #{obj.report.report_id}
                                        </Typography>
                                    )
                                },
                                {
                                    label: 'Status',
                                    tooltip: 'Status',
                                    sortType: 'string',
                                    align: 'left',
                                    virtual: true,
                                    virtualKey: 'status',
                                    width: '200px',
                                    queryValue: (o) =>
                                        reportStatusString(
                                            o.report.report_status
                                        ),
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
                                                            .created_on as never as string
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
                                    queryValue: (o) =>
                                        o.subject.personaname +
                                        o.subject.steam_id,
                                    renderer: (row) => (
                                        <PersonCell
                                            steam_id={row.subject.steam_id}
                                            personaname={
                                                row.subject.personaname
                                            }
                                            avatar={row.subject.avatar}
                                        ></PersonCell>
                                    )
                                },
                                {
                                    label: 'Reporter',
                                    tooltip: 'Reporter',
                                    sortType: 'string',
                                    align: 'left',
                                    queryValue: (o) =>
                                        o.subject.personaname +
                                        o.subject.steam_id,
                                    renderer: (row) => (
                                        <PersonCell
                                            steam_id={row.author.steam_id}
                                            personaname={row.author.personaname}
                                            avatar={row.author.avatar}
                                        ></PersonCell>
                                    )
                                }
                            ]}
                            defaultSortColumn={'report'}
                            rowsPerPage={10}
                            rows={cachedReports}
                        />
                    </Paper>
                </Stack>
            </Grid>
        </Grid>
    );
};
