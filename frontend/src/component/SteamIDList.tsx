import React from 'react';
import FingerprintIcon from '@mui/icons-material/Fingerprint';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemText from '@mui/material/ListItemText';
import { useTheme } from '@mui/material/styles';
import SteamID from 'steamid';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { ContainerWithHeader } from './ContainerWithHeader';

export interface SteamIDListProps {
    steam_id: string;
}

export const SteamIDList = ({ steam_id }: SteamIDListProps) => {
    const theme = useTheme();
    const { sendFlash } = useUserFlashCtx();
    const sid = new SteamID(steam_id);

    if (steam_id == '') {
        return <></>;
    }

    return (
        <ContainerWithHeader
            title={'Steam ID'}
            iconLeft={<FingerprintIcon />}
            marginTop={0}
        >
            <List dense={true}>
                {[
                    [sid.getSteamID64(), 'steam64'],
                    [sid.getSteam3RenderedID(), 'steam3'],
                    [sid.getSteam2RenderedID(true), 'steam2']
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
        </ContainerWithHeader>
    );
};
