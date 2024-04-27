import Typography from '@mui/material/Typography';

export const ErrorComponent = ({ error }: { error: unknown }) => {
    return (
        <>
            <Typography marginTop={3} variant={'h2'} color={'error'} textAlign={'center'}>
                🤯 🤯 🤯 Something went wrong 🤯 🤯 🤯
            </Typography>

            <Typography marginTop={3} variant={'h6'} color={'error'} textAlign={'center'}>{`${error}`}</Typography>
        </>
    );
};
