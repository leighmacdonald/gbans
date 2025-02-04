import Paper from '@mui/material/Paper';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';

export const QueueHelp = () => {
    return (
        <Paper>
            <Grid container>
                <Grid xs={12} padding={2}>
                    <Typography variant={'h2'} padding={1}>
                        Welcome To The Server Queue!
                    </Typography>
                    <Typography variant={'body1'} padding={1}>
                        The primary goal of the queueing system is to help users in seeding a empty server. This does
                        not mean that it can only be used for empty servers, If you wish to join a server only once its
                        18/24, then that will work as well. This will not place all players on the team automatically.
                    </Typography>
                    <Typography variant={'body1'} padding={1}>
                        To queue for servers, simply click on the queue icon for the relevant servers. Once the queue
                        count reaches the minimum required amount of participants, the queue window will popup for
                        players to join.
                    </Typography>
                    <Typography variant={'body1'} padding={1}>
                        The current minimum queue size is <span style={{ fontWeight: 700 }}>4</span>
                    </Typography>
                    <Typography variant={'body1'} padding={1}>
                        There is a audible sound that plays when your queue is ready, be sure to have sound on if you
                        are going to rely on this.
                    </Typography>
                    <Typography variant={'body1'} padding={1}>
                        Players can chat using the queue chat window above. Its accessible though the keyboard icon on
                        the top navigation bar of the site. Note that the same rules as far as language applies here as
                        it does in-game on the servers. Failure to comply with this will have your queue privileges
                        revoked and possibly further actions taken as well.
                    </Typography>
                    <Typography variant={'body1'} padding={1} fontWeight={'bold'} textAlign={'center'}>
                        This functionality requires that you be logged in, and on the same account that you are playing
                        on.
                    </Typography>
                </Grid>
            </Grid>
        </Paper>
    );
};
