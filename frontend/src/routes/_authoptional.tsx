import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/_authoptional')({
    beforeLoad: ({ context }) => {
        // Otherwise, return the user in context
        return context.auth;
    }
});
