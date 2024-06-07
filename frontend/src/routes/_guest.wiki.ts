import { createFileRoute } from '@tanstack/react-router';
import { checkFeatureEnabled } from '../util/features.ts';

export const Route = createFileRoute('/_guest/wiki')({
    beforeLoad: () => {
        checkFeatureEnabled('wiki_enabled');
    }
});
