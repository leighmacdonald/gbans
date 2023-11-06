import React, { useEffect, useMemo, useState, JSX } from 'react';
import { Link as RouterLink, useNavigate } from 'react-router-dom';
import HistoryIcon from '@mui/icons-material/History';
import InfoIcon from '@mui/icons-material/Info';
import VisibilityIcon from '@mui/icons-material/Visibility';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemText from '@mui/material/ListItemText';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { apiGetReports, reportStatusString, ReportWithAuthor } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { DataTable } from '../component/DataTable';
import { PersonCell } from '../component/PersonCell';
import { ReportForm } from '../component/ReportForm';
import { ReportStatusIcon } from '../component/ReportStatusIcon';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { logErr } from '../util/errors';

export const ReportCreatePage = (): JSX.Element => {
    const { currentUser } = useCurrentUserCtx();
    const [reportHistory, setReportHistory] = useState<ReportWithAuthor[]>([]);
    const navigate = useNavigate();

    const canReport = useMemo(() => {
        return currentUser.steam_id && currentUser.ban_id == 0;
    }, [currentUser.ban_id, currentUser.steam_id]);

    useEffect(() => {
        if (canReport) {
            apiGetReports({
                author_id: currentUser.steam_id.toString(),
                limit: 1000,
                order_by: 'created_on',
                desc: true
            })
                .then((resp) => {
                    setReportHistory(resp.data);
                })
                .catch(logErr);
        }
    }, [canReport, currentUser]);

    return (
        <Grid container spacing={3}>
            <Grid xs={12} md={8}>
                <Stack spacing={2}>
                    {canReport && <ReportForm />}
                    {!canReport && (
                        <ContainerWithHeader title={'Permission Denied'}>
                            <Typography variant={'body1'} padding={2}>
                                You are unable to report players while you are
                                currently banned/muted.
                            </Typography>
                            <ButtonGroup sx={{ padding: 2 }}>
                                <Button
                                    variant={'contained'}
                                    color={'primary'}
                                    onClick={() => {
                                        navigate(`/ban/${currentUser.ban_id}`);
                                    }}
                                >
                                    Appeal Ban
                                </Button>
                            </ButtonGroup>
                        </ContainerWithHeader>
                    )}
                    <ContainerWithHeader
                        title={'Your Report History'}
                        iconLeft={<HistoryIcon />}
                    >
                        <DataTable
                            columns={[
                                {
                                    label: 'Status',
                                    tooltip: 'Report Status',
                                    sortKey: 'report_status',
                                    sortable: true,
                                    align: 'left',
                                    queryValue: (o) =>
                                        reportStatusString(o.report_status),
                                    renderer: (obj) => (
                                        <Stack direction={'row'} spacing={1}>
                                            <ReportStatusIcon
                                                reportStatus={obj.report_status}
                                            />
                                            <Typography variant={'body1'}>
                                                {reportStatusString(
                                                    obj.report_status
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
                                            avatar_hash={row.subject.avatarhash}
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
                                                        `/report/${row.report_id}`
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
                            defaultSortColumn={'report_id'}
                            rowsPerPage={10}
                            rows={reportHistory || []}
                        />
                    </ContainerWithHeader>
                </Stack>
            </Grid>
            <Grid xs={12} md={4}>
                <ContainerWithHeader
                    title={'Reporting Guide'}
                    iconLeft={<InfoIcon />}
                >
                    <List>
                        <ListItem>
                            <ListItemText>
                                Once your report is posted, it will be reviewed
                                by a moderator. If further details are required
                                you will be notified about it.
                            </ListItemText>
                        </ListItem>
                        <ListItem>
                            <ListItemText>
                                If you wish to link to a specific SourceTV
                                recording, you can find them listed{' '}
                                <Link component={RouterLink} to={'/stv'}>
                                    here
                                </Link>
                                . Once you find the recording you want, you may
                                select the report icon which will open a new
                                report with the demo attached. From there you
                                will optionally be able to enter a specific tick
                                if you have one.
                            </ListItemText>
                        </ListItem>
                        <ListItem>
                            <ListItemText>
                                Reports that are made in bad faith, or otherwise
                                are considered to be trolling will be closed,
                                and the reporter will be banned.
                            </ListItemText>
                        </ListItem>

                        <ListItem>
                            <ListItemText>
                                Its only possible to open a single report
                                against a particular player. If you wish to add
                                more evidence or discuss further an existing
                                report, please open the existing report and add
                                it by creating a new message there. You can see
                                your current report history below.
                            </ListItemText>
                        </ListItem>
                    </List>
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
};
