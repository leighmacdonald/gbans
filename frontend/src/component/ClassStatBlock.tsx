import React, { JSX } from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import Typography from '@mui/material/Typography';
import { PlayerClass, PlayerClassNames, PlayerStats } from '../api';
import Paper from '@mui/material/Paper';
import Avatar from '@mui/material/Avatar';

export interface ClassStatBlockProps {
    player_class: PlayerClass;
    stats: PlayerStats;
}

export interface InlineStatBlockProps {
    name: string;
    value: string | number;
}

export const InlineStatBlock = ({
    name,
    value
}: InlineStatBlockProps): JSX.Element => {
    return (
        <Grid container spacing={0}>
            <Grid>
                <Typography variant={'h2'}>{name}</Typography>
            </Grid>
            <Grid>
                <Typography variant={'body1'}>{value}</Typography>
            </Grid>
        </Grid>
    );
};

export const ClassStatBlock = ({
    player_class,
    stats
}: ClassStatBlockProps): JSX.Element => {
    return (
        <Paper>
            <Grid container>
                <Grid xs={12}>
                    <Avatar
                        alt={PlayerClassNames[player_class]}
                        src={`../icons/class_${PlayerClassNames[player_class]}.png`}
                    />
                    <Typography variant={'h3'}>
                        {PlayerClassNames[player_class]}
                    </Typography>
                </Grid>
                <Grid container>
                    <Grid xs={3}>
                        <InlineStatBlock name={'Kills'} value={stats.kills} />
                    </Grid>
                </Grid>
            </Grid>
        </Paper>
    );
};
