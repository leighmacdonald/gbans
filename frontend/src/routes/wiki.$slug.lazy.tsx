import { createLazyFileRoute } from '@tanstack/react-router';
import { Wiki } from './wiki.lazy.tsx';

export const Route = createLazyFileRoute('/wiki/$slug')({
    component: Wiki
});
