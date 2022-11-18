import React from 'react';
import { usePugCtx } from './PugCtx';
import Typography from '@mui/material/Typography';
import Button from '@mui/material/Button';
import ForwardIcon from '@mui/icons-material/Forward';
import Card from '@mui/material/Card';
import Box from '@mui/material/Box';
import CardContent from '@mui/material/CardContent';
import CardMedia from '@mui/material/CardMedia';
import { GameType } from '../component/formik/GameTypeField';
import { GameConfig } from '../component/formik/GameConfigField';
import Stack from '@mui/material/Stack';

export const PugLobbyList = () => {
    const { lobbies, joinLobby } = usePugCtx();
    return (
        <Stack marginTop={0} spacing={1}>
            {lobbies.map((lobby) => {
                return (
                    <Card
                        sx={{ display: 'flex' }}
                        key={`lobby-${lobby.lobbyId}`}
                    >
                        <CardMedia
                            component="img"
                            sx={{ width: 150 }}
                            image="https://placekitten.com/150/150"
                            alt="Live from space album cover"
                        />
                        <Box
                            sx={{
                                display: 'flex',
                                flexDirection: 'column',
                                width: '100%'
                            }}
                        >
                            <CardContent sx={{ flex: '1 0 auto' }}>
                                <Typography component="div" variant="h5">
                                    {lobby.options.map_name}
                                </Typography>
                                <Typography
                                    variant="subtitle1"
                                    color="text.secondary"
                                    component="div"
                                >
                                    {`${
                                        lobby.options.game_type ==
                                        GameType.highlander
                                            ? 'Highlander'
                                            : 'Sixes'
                                    } (config: ${
                                        lobby.options.game_config ==
                                        GameConfig.rgl
                                            ? 'RGL'
                                            : 'ETF2L'
                                    })`}
                                </Typography>
                                <Typography variant={'body1'}>
                                    {lobby.options.description}
                                </Typography>
                            </CardContent>
                            <Box
                                sx={{
                                    display: 'flex',
                                    alignItems: 'center',
                                    pl: 1,
                                    pb: 1
                                }}
                            >
                                <Button
                                    startIcon={<ForwardIcon />}
                                    aria-label="join lobby"
                                    variant={'contained'}
                                    onClick={() => {
                                        joinLobby(lobby.lobbyId);
                                    }}
                                >
                                    Join Lobby
                                </Button>
                                <Typography variant={'body1'} marginLeft={2}>
                                    ID: {lobby.lobbyId}
                                </Typography>
                            </Box>
                        </Box>
                    </Card>
                );
            })}
        </Stack>
    );
};
