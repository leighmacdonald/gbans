import { useEffect, useState } from 'react';
import { APIError, apiGetReport, ReportWithAuthor } from '../api';
import { logErr } from '../util/errors.ts';

export const useReport = (report_id: number) => {
    const [data, setData] = useState<ReportWithAuthor>();
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<APIError>();

    useEffect(() => {
        const abortController = new AbortController();
        apiGetReport(report_id, abortController)
            .then((response) => {
                setData(response);
            })
            .catch((e) => {
                logErr(e);
                setError(e);
                return;
            })
            .finally(() => {
                setLoading(false);
            });

        return () => abortController.abort();
    }, []);

    return { data, loading, error };
};
