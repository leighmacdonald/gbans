import { createFileRoute } from '@tanstack/react-router';
import { checkFeatureEnabled } from '../util/features.ts';

export const Route = createFileRoute('/_auth/report')({
    beforeLoad: () => {
        checkFeatureEnabled('reports_enabled');
    }
});
