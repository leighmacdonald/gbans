import React from 'react';
import { UserProfile } from '../api';
import Stack from '@mui/material/Stack';
import Avatar from '@mui/material/Avatar';
import Typography from '@mui/material/Typography';
import Chip from '@mui/material/Chip';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import FunctionsIcon from '@mui/icons-material/Functions';
import PercentIcon from '@mui/icons-material/Percent';
import Container from '@mui/material/Container';
import ButtonGroup from '@mui/material/ButtonGroup';
import Button from '@mui/material/Button';
import ChevronLeftIcon from '@mui/icons-material/ChevronLeft';
import ChevronRightIcon from '@mui/icons-material/ChevronRight';

export interface PlayerClassSelectionProps {
    reverse?: boolean;
    user?: UserProfile;
}

export const PlayerClassSelection = ({
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
