import Avatar from '@mui/material/Avatar';
import Box from '@mui/material/Box';
import ClickAwayListener from '@mui/material/ClickAwayListener';
import Grow from '@mui/material/Grow';
import MenuItem from '@mui/material/MenuItem';
import MenuList from '@mui/material/MenuList';
import Paper from '@mui/material/Paper';
import Popper from '@mui/material/Popper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import React from 'react';
import { useNavigate } from 'react-router-dom';
import { Person, Team } from '../api';
import { createExternalLinks, to } from '../util/history';

export const teamColour = (team: Team): string => {
    switch (team) {
        case Team.BLU:
            return '#99C2D8';
        case Team.RED:
            return '#FB524F';
        default:
            return '#b98e64';
    }
};

export interface PersonaNameLabelProps {
    hideLabel?: boolean;
    source: Person;
    team: Team;
    setFilter: (sid: string) => void;
}

export const ProfileButton = ({
    source,
    team,
    setFilter,
    hideLabel
}: PersonaNameLabelProps) => {
    const navigate = useNavigate();
    const [open, setOpen] = React.useState(false);
    const anchorRef = React.useRef<HTMLButtonElement>(null);

    const handleToggle = () => {
        setOpen((prevOpen) => !prevOpen);
    };

    const handleClose = (event: Event | React.SyntheticEvent) => {
        if (
            anchorRef.current &&
            anchorRef.current.contains(event.target as HTMLElement)
        ) {
            return;
        }

        setOpen(false);
    };

    function handleListKeyDown(event: React.KeyboardEvent) {
        if (event.key === 'Tab') {
            event.preventDefault();
            setOpen(false);
        } else if (event.key === 'Escape') {
            setOpen(false);
        }
    }

    // return focus to the button when we transitioned from !open -> open
    const prevOpen = React.useRef(open);
    React.useEffect(() => {
        if (prevOpen.current && !open) {
            anchorRef.current?.focus();
        }

        prevOpen.current = open;
    }, [open]);

    const theme = useTheme();
    return (
        <Stack
            sx={{
                '&:hover': {
                    backgroundColor: theme.palette.background.default,
                    cursor: 'pointer'
                },
                paddingRight: 2,
                paddingLeft: 2,
                borderRadius: '3px'
            }}
            direction={'row'}
            spacing={1}
            justifyItems={'center'}
            onClick={handleToggle}
        >
            <Box
                sx={{
                    display: 'inline-block',
                    height: '100%',
                    margin: '8px 4px 8px 0'
                }}
                ref={anchorRef}
            >
                <Avatar
                    alt={source.personaname}
                    src={source.avatar}
                    variant={'square'}
                    sx={{
                        verticalAlign: 'middle',
                        height: '24px',
                        width: '24px'
                    }}
                />
            </Box>
            {!hideLabel && (
                <Typography
                    sx={{ overflow: 'hidden', verticalAlign: 'middle' }}
                    variant={'body1'}
                    color={teamColour(team)}
                    lineHeight={3}
                >
                    {source.personaname}
                </Typography>
            )}
            <Popper
                style={{ zIndex: 100 }}
                open={open}
                anchorEl={anchorRef.current}
                role={undefined}
                placement="bottom-start"
                transition
                disablePortal
            >
                {({ TransitionProps, placement }) => (
                    <Grow
                        {...TransitionProps}
                        style={{
                            transformOrigin:
                                placement === 'bottom-start'
                                    ? 'left top'
                                    : 'left bottom'
                        }}
                    >
                        <Paper elevation={2}>
                            <ClickAwayListener onClickAway={handleClose}>
                                <MenuList
                                    autoFocusItem={open}
                                    id="composition-menu"
                                    aria-labelledby="composition-button"
                                    onKeyDown={handleListKeyDown}
                                >
                                    <MenuItem
                                        onClick={() => {
                                            setFilter(source.steam_id);
                                        }}
                                    >
                                        Filter
                                    </MenuItem>
                                    <MenuItem
                                        onClick={() => {
                                            navigate(
                                                `/profile/${source.steam_id}`
                                            );
                                        }}
                                    >
                                        Profile
                                    </MenuItem>
                                    {createExternalLinks(source.steam_id).map(
                                        (l) => {
                                            return (
                                                <MenuItem
                                                    key={l.url}
                                                    onClick={() => {
                                                        to(l.url);
                                                    }}
                                                >
                                                    {l.title}
                                                </MenuItem>
                                            );
                                        }
                                    )}
                                </MenuList>
                            </ClickAwayListener>
                        </Paper>
                    </Grow>
                )}
            </Popper>
        </Stack>
    );
};
