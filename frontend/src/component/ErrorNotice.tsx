import Typography from '@mui/material/Typography';

export const ErrorNotice = ({ error }: { error?: Error }) => {
    return (
        <Typography marginTop={3} variant={'h2'} color={'error'} textAlign={'center'}>
            🤯 🤯 🤯 Something went wrong 🤯 🤯 🤯
            {error?.toString()}
        </Typography>
    );
};
