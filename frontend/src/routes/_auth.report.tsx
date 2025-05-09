import { createFileRoute } from '@tanstack/react-router';
import { ensureFeatureEnabled } from '../util/features.ts';

export const Route = createFileRoute('/_auth/report')({
    beforeLoad: () => {
        ensureFeatureEnabled('reports_enabled');
    }
});
