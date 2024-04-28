import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { createLazyFileRoute } from '@tanstack/react-router';

export const Route = createLazyFileRoute('/_auth/page-not-found')({
    component: PageNotFound
});

// interface PageNotFoundProps {
//     heading?: string;
//     error?: string;
// }

export function PageNotFound() {
    const heading = 'Not Found';
    // const error = null;

    return (
        <Grid container xs={12} padding={2}>
            <Grid xs={12} alignContent={'center'}>
                <Typography align={'center'} variant={'h1'}>
                    {heading}
                </Typography>
                {/*{error && (*/}
                {/*    <Typography align={'center'} variant={'subtitle1'}>*/}
                {/*        {error}*/}
                {/*    </Typography>*/}
                {/*)}*/}
            </Grid>
        </Grid>
    );
}
