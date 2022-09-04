import React from 'react';
import Grid from '@mui/material/Grid';
import Stack from '@mui/material/Stack';
import Avatar from '@mui/material/Avatar';
import Typography from '@mui/material/Typography';
import Chip from '@mui/material/Chip';
import FunctionsIcon from '@mui/icons-material/Functions';
import Paper from '@mui/material/Paper';
import PercentIcon from '@mui/icons-material/Percent';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import { Heading } from '../component/Heading';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { UserProfile } from '../api';
import ChevronLeftIcon from '@mui/icons-material/ChevronLeft';
import ChevronRightIcon from '@mui/icons-material/ChevronRight';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import Container from '@mui/material/Container';
import scoutIcon from '../icons/class_scout.png';
import soldierIcon from '../icons/class_soldier.png';
import pyroIcon from '../icons/class_pyro.png';
import demoIcon from '../icons/class_demoman.png';
import heavyIcon from '../icons/class_heavy.png';
import engyIcon from '../icons/class_engineer.png';
import medicIcon from '../icons/class_medic.png';
import sniperIcon from '../icons/class_sniper.png';
import spyIcon from '../icons/class_spy.png';

interface ClassBoxProps {
    src: string;
}
const ClassBox = ({ src }: ClassBoxProps) => {
    return (
        <Grid sx={{ height: 70 }} container>
            <Grid item xs={12} alignItems="center" alignContent={'center'}>
                <Avatar src={src} sx={{ textAlign: 'center' }} />
            </Grid>
        </Grid>
    );
};

interface PlayerClassSelectionProps {
    reverse?: boolean;
    user?: UserProfile;
}

const PlayerClassSelection = ({
    reverse,
    user
}: PlayerClassSelectionProps): JSX.Element => {
    const height = 70;
    if (user) {
        return (
            <Stack
                height={height}
                direction={reverse ? 'row-reverse' : 'row'}
                padding={1}
                spacing={2}
                paddingLeft={2}
                paddingRight={2}
            >
                <Avatar alt="Remy Sharp" src="/static/images/avatar/1.jpg" />
                <Stack>
                    <Typography
                        variant={'body1'}
                        textAlign={reverse ? 'right' : 'left'}
                    >
                        Player Name
                    </Typography>
                    <Stack direction={reverse ? 'row-reverse' : 'row'}>
                        <Chip icon={<AccessTimeIcon />} label="Hours In Game" />
                        <Chip icon={<FunctionsIcon />} label="Total Games" />
                        <Chip
                            icon={<PercentIcon />}
                            label="Completion Percentage"
                        />
                    </Stack>
                </Stack>
            </Stack>
        );
    } else {
        return (
            <Container sx={{ padding: 2, height: height }}>
                <ButtonGroup
                    fullWidth
                    sx={{ height: '100%' }}
                    variant={'outlined'}
                >
                    <Button
                        {...(!reverse
                            ? { startIcon: <ChevronLeftIcon /> }
                            : { endIcon: <ChevronRightIcon /> })}
                        fullWidth
                    >
                        Join Slot
                    </Button>
                </ButtonGroup>
            </Container>
        );
    }
};

export const Pug = (): JSX.Element => {
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
