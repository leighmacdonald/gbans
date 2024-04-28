import { useEffect } from 'react';
import Typography from '@mui/material/Typography';
import { createLazyFileRoute, useNavigate, useRouteContext } from '@tanstack/react-router';

export const Route = createLazyFileRoute('/_auth/logout')({
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
