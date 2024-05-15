import ErrorIcon from '@mui/icons-material/Error';
import Button from '@mui/material/Button';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { AppError, ErrorCode } from '../error.tsx';
import steamLogo from '../icons/steam_login_lg.png';
import { generateOIDCLink } from '../util/auth/generateOIDCLink.ts';
import { ContainerWithHeader } from './ContainerWithHeader.tsx';

const ErrorBox = ({ error }: { error: string }) => {
    return (
        <Typography variant={'body1'} padding={2} textAlign={'center'}>
            {error}
        </Typography>
    );
};

export const ErrorDetails = ({ error }: { error: AppError | unknown }) => {
    if (error instanceof AppError) {
        return (
            <ContainerWithHeader title={error.name} iconLeft={<ErrorIcon />}>
                {error.code == ErrorCode.LoginRequired ? (
                    <>
                        <ErrorBox error={error.message} />
                        <Stack
                            justifyContent="center"
                            gap={2}
                            flexDirection="row"
                            width={1.0}
                            flexWrap="wrap"
                            padding={2}
                        >
                            <Button
                                sx={{ alignSelf: 'center' }}
                                component={Link}
                                href={generateOIDCLink(window.location.pathname)}
                            >
                                <img src={steamLogo} alt={'Steam Login'} />
                            </Button>
                        </Stack>
                    </>
                ) : (
                    <ErrorBox error={error.message} />
                )}
            </ContainerWithHeader>
        );
    }

    return (
        <ContainerWithHeader title={'Unhandled Error'} iconLeft={<ErrorIcon />}>
            <ErrorBox error={String(error)} />
        </ContainerWithHeader>
    );
};
