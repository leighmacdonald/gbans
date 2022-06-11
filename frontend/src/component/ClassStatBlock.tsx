import React from 'react';
import Grid from '@mui/material/Grid';
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
            <Grid item>
                <Typography variant={'h2'}>{name}</Typography>
            </Grid>
            <Grid item>
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
                <Grid item>
                    <Avatar
                        alt={PlayerClassNames[player_class]}
                        src={`../icons/class_${PlayerClassNames[player_class]}.png`}
                    />
                    <Typography variant={'h3'}>
                        {PlayerClassNames[player_class]}
                    </Typography>
                </Grid>
                <Grid container>
                    <Grid item xs={3}>
                        <InlineStatBlock name={'Kills'} value={stats.kills} />
                    </Grid>
                </Grid>
            </Grid>
        </Paper>
    );
};
