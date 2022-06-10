import React, { useEffect, useState } from 'react';
import Grid from '@mui/material/Grid';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemText from '@mui/material/ListItemText';
import Paper from '@mui/material/Paper';
import ListSubheader from '@mui/material/ListSubheader';
import Stack from '@mui/material/Stack';
import ListItemIcon from '@mui/material/ListItemIcon';
import { ReportForm } from '../component/ReportForm';
import { apiGetReports, ReportWithAuthor } from '../api';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import Link from '@mui/material/Link';
import ListItemAvatar from '@mui/material/ListItemAvatar';
import Avatar from '@mui/material/Avatar';
import { logErr } from '../util/errors';
import { ReportStatusIcon } from '../component/ReportStatusIcon';

type BanState = 'banned' | 'closed';

export interface UserReportHistory {
    name: string;
    target: string;
    target_avatar: string;
    state: BanState;
    updated_on: Date;
    created_on: Date;
}

export const ReportCreatePage = (): JSX.Element => {
    const { currentUser } = useCurrentUserCtx();
    const [reportHistory, setReportHistory] = useState<ReportWithAuthor[]>([]);
    useEffect(() => {
        if (currentUser.steam_id != '') {
            apiGetReports({
                author_id: currentUser.steam_id,
                limit: 10,
                order_by: 'created_on',
                desc: true
            })
                .then((resp) => {
                    resp && setReportHistory(resp);
                })
                .catch(logErr);
        }
    }, [currentUser]);

    return (
        <Grid container spacing={3} paddingTop={3}>
            <Grid item xs={12} xl={8}>
                <Paper elevation={1}>
                    <ReportForm />
                </Paper>
            </Grid>
            <Grid item xs={12} xl={4}>
                <Stack spacing={2}>
                    <Paper elevation={1}>
                        <List
                            subheader={
                                <ListSubheader
                                    component="div"
                                    id="nested-list-subheader"
                                >
                                    Reporting Guide
                                </ListSubheader>
                            }
                        >
                            <ListItem>
                                <ListItemText>
                                    Once your report is posted, it will be
                                    reviewed by an Uncletopia moderator. If
                                    further details are required you will be
                                    notified about it on here.
                                </ListItemText>
                            </ListItem>
                        </List>
                        <List>
                            <ListItem>
                                <ListItemText>
                                    Reports that are made in bad faith, or
                                    otherwise are considered to be trolling will
                                    be closed, and the reporter will be banned
                                    permanently.
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
                                    there.
                                </ListItemText>
                            </ListItem>
                        </List>
                        <List>
                            <ListItem>
                                <ListItemText>
                                    You can see the status of your more recent
                                    reports below.
                                </ListItemText>
                            </ListItem>
                        </List>
                    </Paper>

                    <Paper elevation={1}>
                        <List
                            subheader={
                                <ListSubheader
                                    component="div"
                                    id="nested-list-subheader"
                                >
                                    Your Report History
                                </ListSubheader>
                            }
                        >
                            {reportHistory.map((value, idx) => (
                                <ListItem key={idx}>
                                    <ListItemIcon>
                                        <ReportStatusIcon
                                            reportStatus={
                                                value.report.report_status
                                            }
                                        />
                                    </ListItemIcon>
                                    <ListItemAvatar>
                                        <Avatar
                                            src={value.author.avatar}
                                            variant={'square'}
                                        />
                                    </ListItemAvatar>
                                    <ListItemText>
                                        <Link
                                            href={`/report/${value.report.report_id}`}
                                        >
                                            {value.author.personaname ??
                                                value.author.steam_id}
                                        </Link>
                                    </ListItemText>
                                </ListItem>
                            ))}
                        </List>
                    </Paper>
                </Stack>
            </Grid>
        </Grid>
    );
};
