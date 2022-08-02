import React, { Suspense, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import ListItemButton from '@mui/material/ListItemButton';
import ListItemAvatar from '@mui/material/ListItemAvatar';
import Avatar from '@mui/material/Avatar';
import ListItemText from '@mui/material/ListItemText';
import List from '@mui/material/List';
import Stack from '@mui/material/Stack';
import { Person } from '../api';
import useTheme from '@mui/material/styles/useTheme';
import { LoadingSpinner } from './LoadingSpinner';
import { Heading } from './Heading';
import Pagination from '@mui/material/Pagination';
import SearchIcon from '@mui/icons-material/Search';
import TextField from '@mui/material/TextField';
import CloseIcon from '@mui/icons-material/Close';
import IconButton from '@mui/material/IconButton';

export interface FriendListProps {
    friends: Person[];
    limit?: number;
}

export const FriendList = ({ friends, limit = 25 }: FriendListProps) => {
    const navigate = useNavigate();
    const [searchOpen, setSearchOpen] = useState<boolean>(false);
    const [page, setPage] = useState<number>(0);
    const [query, setQuery] = useState<string>('');

    const filtered = useMemo(() => {
        return friends.filter((friend) => {
            if (friend.personaname.includes(query)) {
                return true;
            } else if (friend.steamid.toString() == query) {
                return true;
            }
            // TODO convert steamids from other formats to query
            return false;
        });
    }, [friends, query]);

    const pages = useMemo(() => {
        return filtered ? Math.floor(filtered.length / limit) : 0;
    }, [filtered, limit]);

    const theme = useTheme();
    return (
        <Stack>
            <Heading>
                {searchOpen ? (
                    <Stack direction={'row'}>
                        <TextField
                            value={query}
                            variant={'standard'}
                            fullWidth
                            onChange={(event) => {
                                setQuery(event.target.value);
                            }}
                        />
                        <IconButton size={'small'}>
                            <CloseIcon
                                onClick={() => {
                                    setSearchOpen(false);
                                }}
                            />
                        </IconButton>
                    </Stack>
                ) : (
                    <Stack direction={'row'} justifyContent={'center'}>
                        <IconButton size={'small'}>
                            <SearchIcon
                                onClick={() => {
                                    setSearchOpen(true);
                                }}
                            />
                        </IconButton>
                        Friends ({friends ? friends.length : 0})
                    </Stack>
                )}
            </Heading>
            <List dense={true}>
                <Suspense fallback={<LoadingSpinner />}>
                    {filtered
                        .slice(page * limit, page * limit + limit)
                        .map((p) => (
                            <ListItemButton
                                color={
                                    p.vac_bans > 0
                                        ? theme.palette.error.main
                                        : undefined
                                }
                                key={`${p.steamid}`}
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
                                    secondary={`${p.steamid}`}
                                />
                            </ListItemButton>
                        ))}
                </Suspense>
            </List>
            <Pagination
                sx={{ width: '100%' }}
                variant={'text'}
                count={pages}
                onChange={(_, newPage) => {
                    setPage(newPage - 1);
                }}
            />
        </Stack>
    );
};
