import React from 'react';
import ListItemText from '@mui/material/ListItemText';
import List from '@mui/material/List';
import Stack from '@mui/material/Stack';
import ListSubheader from '@mui/material/ListSubheader';
import { ListItem } from '@mui/material';
import SteamID from 'steamid';

export interface SteamIDListProps {
    steam_id: string;
}

export const SteamIDList = ({ steam_id }: SteamIDListProps) => {
    const sid = new SteamID(steam_id);
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
                <ListItem>
                    <ListItemText
                        primary={sid.getSteamID64()}
                        secondary={'steam64'}
                    />
                </ListItem>
                <ListItem>
                    <ListItemText
                        primary={sid.getSteam3RenderedID()}
                        secondary={'steam3'}
                    />
                </ListItem>
                <ListItem>
                    <ListItemText
                        primary={sid.getSteam2RenderedID(true)}
                        secondary={'steam2'}
                    />
                </ListItem>
            </List>
        </Stack>
    );
};
