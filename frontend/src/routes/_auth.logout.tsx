import { useEffect } from 'react';
import Typography from '@mui/material/Typography';
import { createFileRoute, useNavigate, useRouteContext } from '@tanstack/react-router';

export const Route = createFileRoute('/_auth/logout')({
    component: LogoutPage
});

function LogoutPage() {
    const navigate = useNavigate();
    const { logout } = useRouteContext({ from: '/_auth/logout' });

    useEffect(() => {
        logout();
        navigate({ to: '/' });
    }, [logout, navigate]);

    return <Typography variant={'h2'}>Logging out...</Typography>;
}
