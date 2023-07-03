import { apiGetSourceBans, sbBanRecord } from '../api';
import React, { useEffect, useState, JSX } from 'react';
import Grid from '@mui/material/Grid';
import Grid2 from '@mui/material/Unstable_Grid2';
import Typography from '@mui/material/Typography';
import Paper from '@mui/material/Paper';

interface SourceBansListProps {
    steam_id: string;
    is_reporter: boolean;
}

export const SourceBansList = ({
    steam_id,
    is_reporter
}: SourceBansListProps): JSX.Element => {
    const [bans, setBans] = useState<sbBanRecord[]>([]);
    useEffect(() => {
        apiGetSourceBans(steam_id).then((resp) => {
            if (resp.result) {
                setBans(resp.result);
            }
        });
    }, [steam_id]);

    if (!bans.length) {
        return <></>;
    }
    console.log(bans);
    return (
        <Paper>
            <Typography variant={'h5'}>
                {is_reporter
                    ? 'Reporter SourceBans History'
                    : 'Suspect SourceBans History'}
            </Typography>
            <Grid2 container>
                {bans.map((ban) => {
                    return (
                        <Grid item xs={2} key={ban.ban_id}>
                            <Typography variant={'caption'}>
                                {ban.created_on} - {ban.site_name} -{' '}
                                {ban.reason}
                            </Typography>
                        </Grid>
                    );
                })}
            </Grid2>
        </Paper>
    );
};
