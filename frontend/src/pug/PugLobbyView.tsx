import React from 'react';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import { Heading } from '../component/Heading';
import { PlayerClassSelection } from './PlayerClassSelection';
import scoutIcon from '../icons/class_scout.png';
import soldierIcon from '../icons/class_soldier.png';
import pyroIcon from '../icons/class_pyro.png';
import demoIcon from '../icons/class_demoman.png';
import heavyIcon from '../icons/class_heavy.png';
import engyIcon from '../icons/class_engineer.png';
import medicIcon from '../icons/class_medic.png';
import sniperIcon from '../icons/class_sniper.png';
import spyIcon from '../icons/class_spy.png';
import Avatar from '@mui/material/Avatar';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';

const ClassBox = ({ src }: { src: string }) => {
    return (
        <Grid sx={{ height: 70 }} container>
            <Grid item xs={12} alignItems="center" alignContent={'center'}>
                <Avatar src={src} sx={{ textAlign: 'center' }} />
            </Grid>
        </Grid>
    );
};

export const PugLobbyView = () => {
    const { currentUser } = useCurrentUserCtx();
    return (
        <Grid container paddingTop={3} spacing={2}>
            <Grid item xs={12}></Grid>
            <Grid item xs={5}>
                <Paper>
                    <Stack spacing={1}>
                        <Heading>RED</Heading>
                        <PlayerClassSelection reverse user={currentUser} />
                        <PlayerClassSelection reverse user={currentUser} />
                        <PlayerClassSelection reverse />
                        <PlayerClassSelection reverse user={currentUser} />
                        <PlayerClassSelection reverse />
                        <PlayerClassSelection reverse user={currentUser} />
                        <PlayerClassSelection reverse />
                        <PlayerClassSelection reverse user={currentUser} />
                        <PlayerClassSelection reverse user={currentUser} />
                    </Stack>
                </Paper>
            </Grid>
            <Grid item xs={2}>
                <Stack spacing={1}>
                    <Heading>Class</Heading>
                    <ClassBox src={scoutIcon} />
                    <ClassBox src={soldierIcon} />
                    <ClassBox src={pyroIcon} />
                    <ClassBox src={demoIcon} />
                    <ClassBox src={heavyIcon} />
                    <ClassBox src={engyIcon} />
                    <ClassBox src={medicIcon} />
                    <ClassBox src={sniperIcon} />
                    <ClassBox src={spyIcon} />
                </Stack>
            </Grid>
            <Grid item xs={5}>
                <Paper>
                    <Stack spacing={1}>
                        <Heading bgColor={'#395c78'}>BLU</Heading>
                        <PlayerClassSelection user={currentUser} />
                        <PlayerClassSelection />
                        <PlayerClassSelection user={currentUser} />
                        <PlayerClassSelection user={currentUser} />
                        <PlayerClassSelection user={currentUser} />
                        <PlayerClassSelection />
                        <PlayerClassSelection user={currentUser} />
                        <PlayerClassSelection />
                        <PlayerClassSelection user={currentUser} />
                    </Stack>
                </Paper>
            </Grid>
        </Grid>
    );
};
