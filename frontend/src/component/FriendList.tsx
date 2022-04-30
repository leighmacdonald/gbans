import React, { Suspense, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import ListItemButton from '@mui/material/ListItemButton';
import ListItemAvatar from '@mui/material/ListItemAvatar';
import Avatar from '@mui/material/Avatar';
import ListItemText from '@mui/material/ListItemText';
import List from '@mui/material/List';
import Stack from '@mui/material/Stack';
import ButtonGroup from '@mui/material/ButtonGroup';
import ArrowLeft from '@mui/icons-material/ArrowLeft';
import ArrowRight from '@mui/icons-material/ArrowRight';
import Button from '@mui/material/Button';
import ListSubheader from '@mui/material/ListSubheader';
import { Person } from '../api';
import { useTheme } from '@mui/material';
import { LoadingSpinner } from './LoadingSpinner';

export interface FriendListProps {
    friends: Person[];
    limit?: number;
}

export const FriendList = ({ friends, limit = 25 }: FriendListProps) => {
    const navigate = useNavigate();
    const [page, setPage] = useState<number>(0);
    const pages = friends ? Math.floor(friends.length / limit) : 0;
    const nav = (
        <ButtonGroup fullWidth>
            <Button
                variant={'text'}
                onClick={() => {
                    if (page > 0) {
                        setPage(page - 1);
                    }
                }}
            >
                <ArrowLeft />
                Prev
            </Button>
            <Button
                variant={'text'}
                onClick={() => {
                    if (page < pages) {
                        setPage(page + 1);
                    }
                }}
            >
                Next
                <ArrowRight />
            </Button>
        </ButtonGroup>
    );
    const theme = useTheme();
    return (
        <Stack>
            <List
                dense={true}
                subheader={
                    <ListSubheader component="div" id="nested-list-subheader">
                        Friends ({friends ? friends.length : 0})
                    </ListSubheader>
                }
            >
                <Suspense fallback={<LoadingSpinner />}>
                    {(friends || [])
                        .slice(page * limit, page * limit + limit)
                        .map((p) => (
                            <ListItemButton
                                color={
                                    p.vac_bans > 0
                                        ? theme.palette.error.main
                                        : undefined
                                }
                                key={p.steamid}
                                onClick={() => {
                                    navigate(`/profile/${p.steamid}`);
                                }}
                            >
                                <ListItemAvatar>
                                    <Avatar
                                        alt={'Profile Picture'}
                                        src={p.avatar}
                                    />
                                </ListItemAvatar>
                                <ListItemText
                                    primary={p.personaname}
                                    secondary={p.steamid}
                                />
                            </ListItemButton>
                        ))}
                </Suspense>
            </List>
            {nav}
        </Stack>
    );
};
