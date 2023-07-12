import React, { useState } from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import { PugCreateLobbyForm } from './PugCreateLobbyForm';
import Typography from '@mui/material/Typography';
import CardActions from '@mui/material/CardActions';
import Card from '@mui/material/Card';
import CardMedia from '@mui/material/CardMedia';
import CardContent from '@mui/material/CardContent';
import { LoadingButton } from '@mui/lab';
import { PugLobbyList } from './PugLobbyList';
import AddIcon from '@mui/icons-material/Add';

export const JoinOrCreateLobby = ({ isReady }: { isReady: boolean }) => {
    const [open, setOpen] = useState(false);

    return (
        <>
            <PugCreateLobbyForm setOpen={setOpen} open={open} />
            <Grid
                container
                direction="row"
                justifyContent="space-around"
                alignItems={'start'}
                paddingTop={3}
                spacing={2}
            >
                <Grid xs={8}>
                    <PugLobbyList />
                </Grid>
                <Grid xs={4}>
                    <Card sx={{ maxWidth: 350 }}>
                        <CardMedia
                            component="img"
                            height="200"
                            image={'https://placekitten.com/200/350'}
                            alt="kitty"
                        />
                        <CardContent>
                            <Typography
                                gutterBottom
                                variant="h5"
                                component="div"
                            >
                                Create New Lobby
                            </Typography>
                            <Typography variant="body2" color="text.secondary">
                                Lizards are a widespread group of squamate
                                reptiles, with over 6,000 species, ranging
                                across all continents except Antarctica
                            </Typography>
                        </CardContent>
                        <CardActions>
                            <LoadingButton
                                startIcon={<AddIcon />}
                                variant={'contained'}
                                size="small"
                                title={'Loading...'}
                                loading={!isReady}
                                disabled={open || !isReady}
                                onClick={() => {
                                    setOpen(true);
                                }}
                            >
                                Create Lobby
                            </LoadingButton>
                        </CardActions>
                    </Card>
                </Grid>
            </Grid>
        </>
    );
};
