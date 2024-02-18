import { Suspense, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import CloseIcon from '@mui/icons-material/Close';
import SearchIcon from '@mui/icons-material/Search';
import Avatar from '@mui/material/Avatar';
import Container from '@mui/material/Container';
import IconButton from '@mui/material/IconButton';
import List from '@mui/material/List';
import ListItemAvatar from '@mui/material/ListItemAvatar';
import ListItemButton from '@mui/material/ListItemButton';
import ListItemText from '@mui/material/ListItemText';
import Pagination from '@mui/material/Pagination';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import { Person } from '../api';
import { avatarHashToURL, filterPerson } from '../util/text.tsx';
import { Heading } from './Heading';
import { LoadingSpinner } from './LoadingSpinner';

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
        return filterPerson(friends, query);
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
                        <IconButton
                            size={'small'}
                            onClick={() => {
                                setSearchOpen(false);
                            }}
                        >
                            <CloseIcon />
                        </IconButton>
                    </Stack>
                ) : (
                    <Stack direction={'row'} justifyContent={'center'}>
                        <IconButton
                            size={'small'}
                            onClick={() => {
                                setSearchOpen(true);
                            }}
                        >
                            <SearchIcon />
                        </IconButton>
                        Friends ({friends ? friends.length : 0})
                    </Stack>
                )}
            </Heading>
            <List dense={true}>
                <Suspense fallback={<LoadingSpinner />}>
                    {friends.length == 0 && (
                        <Container>
                            <Typography textAlign={'center'} variant={'body2'}>
                                😢
                            </Typography>
                        </Container>
                    )}
                    {friends.length > 0 &&
                        filtered
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
                                            src={avatarHashToURL(p.avatarhash)}
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
            {friends.length > 0 && (
                <Pagination
                    disabled={friends.length == 0}
                    sx={{ width: '100%' }}
                    variant={'text'}
                    count={pages}
                    onChange={(_, newPage) => {
                        setPage(newPage - 1);
                    }}
                />
            )}
        </Stack>
    );
};
