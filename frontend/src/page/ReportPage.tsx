import React from 'react';
import Grid from '@mui/material/Grid';
import List from '@mui/material/List';
import ListItemText from '@mui/material/ListItemText';
import ListItem from '@mui/material/ListItem';
import Paper from '@mui/material/Paper';
import ListSubheader from '@mui/material/ListSubheader';
import Stack from '@mui/material/Stack';
import { ReportComponent } from '../component/ReportComponent';

export const ReportPage = (): JSX.Element => {
    return (
        <Grid container spacing={3} padding={3}>
            <Grid item xs={6}>
                <Paper elevation={1}>
                    <ReportComponent />
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
                        />
                    </Paper>
                </Stack>
            </Grid>
        </Grid>
    );
};
