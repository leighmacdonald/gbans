import React from 'react';
import ListItemText from '@mui/material/ListItemText';
import List from '@mui/material/List';
import Stack from '@mui/material/Stack';
import ListSubheader from '@mui/material/ListSubheader';
import { ListItem, useTheme } from '@mui/material';
import SteamID from 'steamid';

export interface SteamIDListProps {
    steam_id: bigint;
}

export const SteamIDList = ({ steam_id }: SteamIDListProps) => {
    const sid = new SteamID(steam_id);
    const theme = useTheme();
    return (
        <Stack>
            <List
                dense={true}
                subheader={
                    <ListSubheader component="div" id="steam_id-list-subheader">
                        Steam ID
                    </ListSubheader>
                }
            >
                {[
                    [sid.getSteamID64(), 'steam64'],
                    [sid.getSteam3RenderedID(), 'steam3'],
                    [sid.getSteam2RenderedID(true), 'steam2']
                ].map((s) => {
                    return (
                        <ListItem
                            onClick={async () => {
                                await navigator.clipboard.writeText(s[0]);
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
