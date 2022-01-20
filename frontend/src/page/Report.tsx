import React from 'react';
import Grid from '@mui/material/Grid';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemText from '@mui/material/ListItemText';
import Paper from '@mui/material/Paper';
import { ReportForm } from '../component/ReportForm';
import ListSubheader from '@mui/material/ListSubheader';

export const Report = (): JSX.Element => {
    return (
        <Grid container spacing={3} padding={3}>
            <Grid item xs={6}>
                <Paper elevation={1}>
                    <ReportForm />
                </Paper>
            </Grid>
            <Grid item xs={6}>
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
                                Once your appeal is posted, your appeal will be
                                reviewed by an Uncletopia moderator.
                            </ListItemText>
                        </ListItem>
                    </List>
                    <List>
                        <ListItem>
                            <ListItemText>
                                Hostile or inappropriate messages will be
                                ignored, and you may be subject to removal from
                                the Discord server as well.
                            </ListItemText>
                        </ListItem>
                    </List>
                    <List>
                        <ListItem>
                            <ListItemText>
                                If your appeal involves trying to blame other
                                people who reported you, or other
                                &quot;whataboutisms&quot;, rethink your
                                approach.
                            </ListItemText>
                        </ListItem>
                    </List>
                    <List>
                        <ListItem>
                            <ListItemText>
                                Appeals that we deem are argued in bad faith
                                will also be permanently banned.
                            </ListItemText>
                        </ListItem>
                    </List>
                </Paper>
            </Grid>
        </Grid>
    );
};
