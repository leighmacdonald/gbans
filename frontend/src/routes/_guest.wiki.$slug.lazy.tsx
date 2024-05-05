import { createLazyFileRoute } from '@tanstack/react-router';
import { WikiPage } from '../component/WikiPage.tsx';

export const Route = createLazyFileRoute('/_guest/wiki/$slug')({
    component: Wiki
});

function Wiki() {
    const { slug } = Route.useParams();

    return <WikiPage slug={slug} path={'/_guest/wiki/$slug'} />;
}
