import { createLazyFileRoute } from '@tanstack/react-router';
import { WikiPage } from '../component/WikiPage.tsx';

export const Route = createLazyFileRoute('/_guest/wiki')({
    component: Wiki
});

function Wiki() {
    return <WikiPage slug={'home'} />;
}
