import Grid from '@mui/material/Unstable_Grid2';
import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/_guest/patreon')({
    component: PatreonLazy
});

function PatreonLazy() {
    return (
        <Grid container spacing={2}>
            <Grid></Grid>
        </Grid>
    );
}
