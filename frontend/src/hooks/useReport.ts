import { useEffect, useState } from 'react';
import { apiGetReport, ReportWithAuthor } from '../api';
import { AppError } from '../error.tsx';
import { logErr } from '../util/errors.ts';

export const useReport = (report_id: number) => {
    const [data, setData] = useState<ReportWithAuthor>();
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<AppError>();

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
    }, [report_id]);

    return { data, loading, error };
};
