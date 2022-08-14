import React from 'react';
import ListItemText from '@mui/material/ListItemText';
import List from '@mui/material/List';
import Stack from '@mui/material/Stack';
import useTheme from '@mui/material/styles/useTheme';
import ListItem from '@mui/material/ListItem';
import SteamID from 'steamid';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { Heading } from './Heading';

export interface SteamIDListProps {
    steam_id: SteamID;
}

export const SteamIDList = ({ steam_id }: SteamIDListProps) => {
    const theme = useTheme();
    const { sendFlash } = useUserFlashCtx();
    return (
        <Stack>
            <Heading>Steam ID</Heading>
            <List dense={true}>
                {[
                    [steam_id.getSteamID64(), 'steam64'],
                    [steam_id.getSteam3RenderedID(), 'steam3'],
                    [steam_id.getSteam2RenderedID(true), 'steam2']
                ].map((s) => {
                    return (
                        <ListItem
                            onClick={async () => {
                                await navigator.clipboard.writeText(s[0]);
                                sendFlash(
                                    'success',
                                    `Copied to clipboard: ${s[0]}`
                                );
                            }}
                            key={s[0]}
                            sx={{
                                '&:hover': {
                                    backgroundColor:
                                        theme.palette.background.default,
                                    cursor: 'pointer'
                                }
                            }}
                        >
                            <ListItemText primary={s[0]} secondary={s[1]} />
                        </ListItem>
                    );
                })}
            </List>
        </Stack>
    );
};
