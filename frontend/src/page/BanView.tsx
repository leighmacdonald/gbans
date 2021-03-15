import React, { useEffect } from 'react';
import { useParams } from 'react-router';
import { apiGetBan, BannedPerson } from '../util/api';
import { NotNull } from '../util/types';
import { Grid, Typography } from '@material-ui/core';

interface BanViewParams {
    ban_id: string;
}

export const BanView = (): JSX.Element => {
    const [loading, setLoading] = React.useState<boolean>(true);
    const [ban, setBan] = React.useState<NotNull<BannedPerson>>();
    const { ban_id } = useParams<BanViewParams>();
    useEffect(() => {
        const loadBan = async () => {
            try {
                setBan((await apiGetBan(parseInt(ban_id))) as BannedPerson);
                setLoading(false);
            } catch (e) {
                alert(`Failed to load ban: ${e}`);
            }
        };
        loadBan();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);
    return (
        <Grid container>
            {loading && !ban && (
                <Grid item xs>
                    <Typography variant={'h2'}>Loading profile...</Typography>
                </Grid>
            )}
            {!loading && ban && (
                <Grid container>
                    <Grid item xs={6}>
                        <figure>
                            <img
                                src={ban.person.avatarfull}
                                alt={'Player avatar'}
                            />
                            <figcaption>{ban.person.personaname}</figcaption>
                        </figure>
                    </Grid>
                    <Grid item xs={6} />
                    <Grid item xs>
                        <Typography variant={'h3'}>Chat Logs</Typography>
                    </Grid>
                    {ban?.history_chat &&
                        ban?.history_chat.map((value, i) => {
                            return (
                                <Grid
                                    item
                                    className={'cell'}
                                    key={`chat-log-${i}`}
                                >
                                    <span>{value}</span>
                                </Grid>
                            );
                        })}
                </Grid>
            )}
        </Grid>
    );
};
