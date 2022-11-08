import React, { useState } from 'react';
import Grid from '@mui/material/Grid';
import Button from '@mui/material/Button';
import { PugCreateLobbyForm } from './PugCreateLobbyForm';
import Typography from '@mui/material/Typography';
import CardActions from '@mui/material/CardActions';
import Card from '@mui/material/Card';
import CardMedia from '@mui/material/CardMedia';
import CardContent from '@mui/material/CardContent';

export const JoinOrCreateLobby = () => {
    const [open, setOpen] = useState(false);

    return (
        <>
            <PugCreateLobbyForm setOpen={setOpen} open={open} />
            <Grid
                container
                direction="row"
                justifyContent="space-around"
                alignItems="center"
                paddingTop={3}
                spacing={2}
            >
                <Grid item>
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
                                Join Existing Lobby
                            </Typography>
                            <Typography variant="body2" color="text.secondary">
                                Lizards are a widespread group of squamate
                                reptiles, with over 6,000 species, ranging
                                across all continents except Antarctica
                            </Typography>
                        </CardContent>
                        <CardActions>
                            <Button size="small">Join Lobby</Button>
                        </CardActions>
                    </Card>
                </Grid>
                <Grid item>
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
                            <Button
                                size="small"
                                disabled={open}
                                onClick={() => {
                                    setOpen(true);
                                }}
                            >
                                Create Lobby
                            </Button>
                        </CardActions>
                    </Card>
                </Grid>
            </Grid>
        </>
    );
};
