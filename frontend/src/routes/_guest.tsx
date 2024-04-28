import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/_guest')({
    beforeLoad: ({ context }) => {
        // Otherwise, return the user in context
        return context.auth;
    }
});
