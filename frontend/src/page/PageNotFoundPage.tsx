import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';

interface PageNotFoundProps {
    heading?: string;
    error?: string;
}

export const PageNotFoundPage = ({
    error,
    heading = 'Not Found'
}: PageNotFoundProps) => {
    return (
        <Grid container xs={12} padding={2}>
            <Grid xs={12} alignContent={'center'}>
                <Typography align={'center'} variant={'h1'}>
                    {heading}
                </Typography>
                {error && (
                    <Typography align={'center'} variant={'subtitle1'}>
                        {error}
                    </Typography>
                )}
            </Grid>
        </Grid>
    );
};

export default PageNotFoundPage;
