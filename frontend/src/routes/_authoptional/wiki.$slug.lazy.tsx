import { createLazyFileRoute } from '@tanstack/react-router';
import { Wiki } from './wiki.lazy.tsx';

export const Route = createLazyFileRoute('/_authoptional/wiki/$slug')({
    component: Wiki
});
