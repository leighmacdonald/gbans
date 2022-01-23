import React, { useEffect, useState } from 'react';
import Grid from '@mui/material/Grid';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemText from '@mui/material/ListItemText';
import Paper from '@mui/material/Paper';
import { ReportForm } from '../component/ReportForm';
import ListSubheader from '@mui/material/ListSubheader';
import Stack from '@mui/material/Stack';
import { ListItemAvatar, ListItemIcon } from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import GavelIcon from '@mui/icons-material/Gavel';
import Avatar from '@mui/material/Avatar';

type BanState = 'banned' | 'closed';

export interface UserReportHistory {
    name: string;
    target: string;
    target_avatar: string;
    state: BanState;
    updated_on: Date;
    created_on: Date;
}

export const Report = (): JSX.Element => {
    const [reportHistory, setReportHistory] = useState<UserReportHistory[]>([]);
    useEffect(() => {
        setReportHistory([
            {
                created_on: new Date(),
                updated_on: new Date(),
                state: 'banned',
                name: 'Test Report Subject',
                target: '76561198057999536',
                target_avatar:
                    'https://cdn.akamai.steamstatic.com/steamcommunity/public/images/avatars/8e/8e142e79042c28ce1ac4b59e2262dccee24713e2_full.jpg'
            },
            {
                created_on: new Date(),
                updated_on: new Date(),
                state: 'closed',
                name: 'Test Report Subject #2',
                target: '76561198057999536',
                target_avatar:
                    'https://cdn.akamai.steamstatic.com/steamcommunity/public/images/avatars/8e/8e142e79042c28ce1ac4b59e2262dccee24713e2_full.jpg'
            },
            {
                created_on: new Date(),
                updated_on: new Date(),
                state: 'closed',
                name: 'The bots are invading help us!!!',
                target: '76561198072115209',
                target_avatar:
                    'https://cdn.akamai.steamstatic.com/steamcommunity/public/images/avatars/e3/e3247f3517d5cea98d8ec42ccd8f4c1d6e012e28_full.jpg'
            }
        ]);
    }, []);

    return (
        <Grid container spacing={3} padding={3}>
            <Grid item xs={6}>
                <Paper elevation={1}>
                    <ReportForm />
                </Paper>
            </Grid>
            <Grid item xs={6}>
                <Stack>
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
                                <ListItemText>Some more stuff..</ListItemText>
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
                                        {value.state == 'banned' ? (
                                            <GavelIcon />
                                        ) : (
                                            <CloseIcon />
                                        )}
                                    </ListItemIcon>
                                    <ListItemAvatar>
                                        <Avatar
                                            src={value.target_avatar}
                                            variant={'square'}
                                        />
                                    </ListItemAvatar>
                                    <ListItemText>{value.name}</ListItemText>
                                </ListItem>
                            ))}
                        </List>
                    </Paper>
                </Stack>
            </Grid>
        </Grid>
    );
};
