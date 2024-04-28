import { useEffect, useState } from 'react';
import { PermissionLevel } from '../api';
import { apiGetWikiPage, Page } from '../api/wiki';
import { AppError } from '../error.tsx';
import { logErr } from '../util/errors';

const defaultPage: Page = {
    slug: '',
    body_md: '',
    created_on: new Date(),
    updated_on: new Date(),
    revision: 0,
    title: '',
    permission_level: PermissionLevel.Guest
};

export const useWiki = (slug: string = 'home') => {
    const [data, setData] = useState<Page>(defaultPage);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<AppError>();

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        apiGetWikiPage(slug, abortController)
            .then((resp) => {
                setData(resp);
            })
            .catch((reason) => {
                setError(reason);
                logErr(reason);
            })
            .finally(() => {
                setLoading(false);
            });

        return () => abortController.abort();
    }, [slug]);

    return { data, loading, error };
};
