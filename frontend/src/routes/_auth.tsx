import { createFileRoute, redirect } from '@tanstack/react-router';

export const Route = createFileRoute('/_auth')({
    beforeLoad: ({ context, location }) => {
        // If the user is logged out, redirect them to the login page
        if (!context.auth.isAuthenticated()) {
            throw redirect({
                to: '/login',
                search: {
                    // Use the current location to power a redirect after login
                    // (Do not use `router.state.resolvedLocation` as it can
                    // potentially lag behind the actual current location)
                    redirect: location.href
                }
            });
        }

        // Otherwise, return the user in context
        return context.auth;
    }
});
