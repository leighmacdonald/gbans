import { queryOptions } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import { PermissionLevel } from '../api';
import { apiGetWikiPage, Page } from '../api/wiki.ts';
import { WikiPage } from '../component/WikiPage.tsx';

export const Route = createFileRoute('/_guest/wiki/')({
    component: Wiki,
    loader: async ({ context, abortController }) => {
        const queryOpts = queryOptions({
            queryKey: ['wiki', { slug: 'home' }],
            queryFn: async () => {
                try {
                    return await apiGetWikiPage('home', abortController);
                } catch (e) {
                    return {
                        revision: 0,
                        body_md: '',
                        slug: 'home',
                        permission_level: PermissionLevel.Guest
                    } as Page;
                }
            }
        });

        return context.queryClient.fetchQuery(queryOpts);
    }
});

function Wiki() {
    return <WikiPage slug={'home'} path={'/_guest/wiki/'} />;
}
