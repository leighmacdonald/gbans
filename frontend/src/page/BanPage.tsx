import React, { useEffect } from 'react';
import Grid from '@mui/material/Grid';
import { useParams } from 'react-router-dom';
import { apiGetBan, BannedPerson, BanReasons } from '../api';
import { NotNull } from '../util/types';
import { Heading } from '../component/Heading';
import { SteamIDList } from '../component/SteamIDList';
import { ProfileInfoBox } from '../component/ProfileInfoBox';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import ListItem from '@mui/material/ListItem';
import ListItemText from '@mui/material/ListItemText';
import List from '@mui/material/List';

export const BanPage = (): JSX.Element => {
    //const [loading, setLoading] = React.useState<boolean>(true);
    const [ban, setBan] = React.useState<NotNull<BannedPerson>>();
    const { ban_id } = useParams();

    useEffect(() => {
        apiGetBan(parseInt(ban_id ?? '0') || 0)
            .then((banPerson) => {
                if (banPerson) {
                    setBan(banPerson);
                }
                //setLoading(false);
            })
            .catch((e) => {
                alert(`Failed to load ban: ${e}`);
            });

        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);
    return (
        <Grid container paddingTop={3} spacing={3}>
            <Grid item xs={6}>
                <Stack>
                    <Heading>Comments / Appeal</Heading>
                </Stack>
            </Grid>
            <Grid item xs={6}>
                <Grid container spacing={3}>
                    <Grid item xs={12}>
                        {ban && (
                            <ProfileInfoBox
                                profile={{ player: ban?.person, friends: [] }}
                            />
                        )}
                    </Grid>
                    <Grid item xs={5}>
                        {ban && (
                            <Paper elevation={1}>
                                <SteamIDList steam_id={ban?.ban.steam_id} />
                            </Paper>
                        )}
                    </Grid>
                    <Grid item xs={7}>
                        {ban && (
                            <Paper elevation={1}>
                                <Stack>
                                    <Heading>Ban Details</Heading>
                                    <List dense={true}>
                                        <ListItem>
                                            <ListItemText
                                                primary={'Reason'}
                                                secondary={
                                                    BanReasons[ban.ban.reason]
                                                }
                                            />
                                        </ListItem>
                                        {ban.ban.reason_text != '' && (
                                            <ListItem>
                                                <ListItemText
                                                    primary={'Reason (Custom)'}
                                                    secondary={
                                                        ban.ban.reason_text
                                                    }
                                                />
                                            </ListItem>
                                        )}
                                        <ListItem>
                                            <ListItemText
                                                primary={'Created On'}
                                                secondary={
                                                    ban.ban
                                                        .created_on as any as string
                                                }
                                            />
                                        </ListItem>
                                        <ListItem>
                                            <ListItemText
                                                primary={'Expires'}
                                                secondary={
                                                    ban.ban
                                                        .valid_until as any as string
                                                }
                                            />
                                        </ListItem>
                                    </List>
                                </Stack>
                            </Paper>
                        )}
                    </Grid>
                </Grid>
            </Grid>
        </Grid>
    );
};
