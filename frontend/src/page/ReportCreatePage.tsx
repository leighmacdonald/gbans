import React, { useEffect, useState } from 'react';
import Grid from '@mui/material/Grid';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemText from '@mui/material/ListItemText';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import { ReportForm } from '../component/ReportForm';
import { apiGetReports, reportStatusString, ReportWithAuthor } from '../api';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { logErr } from '../util/errors';
import { ReportStatusIcon } from '../component/ReportStatusIcon';
import { Heading } from '../component/Heading';
import { DataTable } from '../component/DataTable';
import { PersonCell } from '../component/PersonCell';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import VisibilityIcon from '@mui/icons-material/Visibility';
import { useNavigate } from 'react-router-dom';
import Typography from '@mui/material/Typography';

export const ReportCreatePage = (): JSX.Element => {
    const { currentUser } = useCurrentUserCtx();
    const [reportHistory, setReportHistory] = useState<ReportWithAuthor[]>([]);
    const navigate = useNavigate();

    useEffect(() => {
        if (currentUser.steam_id.isValidIndividual()) {
            apiGetReports({
                author_id: currentUser.steam_id.toString(),
                limit: 1000,
                order_by: 'created_on',
                desc: true
            })
                .then((resp) => {
                    if (!resp.status || !resp.result) {
                        return;
                    }
                    setReportHistory(resp.result);
                })
                .catch(logErr);
        }
    }, [currentUser]);

    return (
        <Grid container spacing={3} paddingTop={3}>
            <Grid item xs={12} md={8}>
                <Stack spacing={2}>
                    <Paper elevation={1}>
                        <ReportForm />
                    </Paper>
                    <Paper elevation={1}>
                        <Heading>Your Report History</Heading>
                        <DataTable
                            columns={[
                                {
                                    label: 'Status',
                                    tooltip: 'Report Status',
                                    sortKey: 'report',
                                    sortable: true,
                                    align: 'left',
                                    queryValue: (o) =>
                                        reportStatusString(
                                            o.report.report_status
                                        ),
                                    renderer: (obj) => (
                                        <Stack direction={'row'} spacing={1}>
                                            <ReportStatusIcon
                                                reportStatus={
                                                    obj.report.report_status
                                                }
                                            />
                                            <Typography variant={'body1'}>
                                                {reportStatusString(
                                                    obj.report.report_status
                                                )}
                                            </Typography>
                                        </Stack>
                                    )
                                },
                                {
                                    label: 'Player',
                                    tooltip: 'Reported Player',
                                    sortKey: 'subject',
                                    sortable: true,
                                    align: 'left',
                                    queryValue: (o) =>
                                        `${o.subject.steam_id} ${o.subject.personaname}`,
                                    renderer: (row) => (
                                        <PersonCell
                                            steam_id={row.subject.steam_id}
                                            personaname={
                                                row.subject.personaname
                                            }
                                            avatar={row.subject.avatar}
                                        />
                                    )
                                },
                                {
                                    label: 'View',
                                    tooltip: 'View your report',
                                    sortable: false,
                                    virtual: true,
                                    virtualKey: 'actions',
                                    align: 'right',
                                    renderer: (row) => (
                                        <ButtonGroup>
                                            <IconButton
                                                color={'primary'}
                                                onClick={() => {
                                                    navigate(
                                                        `/report/${row.report.report_id}`
                                                    );
                                                }}
                                            >
                                                <Tooltip title={'View'}>
                                                    <VisibilityIcon />
                                                </Tooltip>
                                            </IconButton>
                                        </ButtonGroup>
                                    )
                                }
                            ]}
                            defaultSortColumn={'report'}
                            rowsPerPage={10}
                            rows={reportHistory || []}
                        />
                    </Paper>
                </Stack>
            </Grid>
            <Grid item xs={12} md={4}>
                <Stack spacing={2}>
                    <Paper elevation={1}>
                        <Heading> Reporting Guide</Heading>
                        <List>
                            <ListItem>
                                <ListItemText>
                                    Once your report is posted, it will be
                                    reviewed by an Uncletopia moderator. If
                                    further details are required you will be
                                    notified about it.
                                </ListItemText>
                            </ListItem>
                        </List>
                        <List>
                            <ListItem>
                                <ListItemText>
                                    Reports that are made in bad faith, or
                                    otherwise are considered to be trolling will
                                    be closed, and the reporter will be banned.
                                </ListItemText>
                            </ListItem>
                        </List>
                        <List>
                            <ListItem>
                                <ListItemText>
                                    Its only possible to open a single report
                                    against a particular player. If you wish to
                                    add more evidence or discus further an
                                    existing report, please open the existing
                                    report and add it by creating a new message
                                    there. You can see your current report
                                    history below.
                                </ListItemText>
                            </ListItem>
                        </List>
                    </Paper>
                </Stack>
            </Grid>
        </Grid>
    );
};
