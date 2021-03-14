import React from 'react';
import {PlayerBanForm} from '../component/PlayerBanForm';
import {Card, CardContent, Grid} from '@material-ui/core';
import {makeStyles} from '@material-ui/core/styles';
import {ProfilePanel} from '../component/ProfilePanel';

const useStyles = makeStyles({
    root: {
        minWidth: 275
    },
    bullet: {
        display: 'inline-block',
        margin: '0 2px',
        transform: 'scale(0.8)'
    },
    title: {
        fontSize: 14
    },
    pos: {
        marginBottom: 12
    }
});

export const AdminBan = () => {
    const classes = useStyles();
    return (
        <Grid container>
            <Grid item xs={6}>
                <PlayerBanForm />
            </Grid>
            <Grid item xs={6}>
                <Card className={classes.root}>
                    <CardContent>
                        <ProfilePanel />
                    </CardContent>
                </Card>
            </Grid>
        </Grid>
    );
};
