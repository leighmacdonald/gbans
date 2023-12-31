import { useEffect, useState } from 'react';
import { apiGetReportMessages, ReportMessage } from '../api';
import { logErr } from '../util/errors';

export const useReportMessages = (report_id: number) => {
    const [loading, setLoading] = useState(false);
    const [data, setData] = useState<ReportMessage[]>([]);
    const [error, setError] = useState();

    useEffect(() => {
        const abortController = new AbortController();
        if (report_id <= 0) {
            return;
        }

        setLoading(true);
        apiGetReportMessages(report_id)
            .then((messages) => {
                setData(messages);
            })
            .catch((reason) => {
                logErr(reason);
                setError(error);
            })
            .finally(() => {
                setLoading(false);
            });

        return () => abortController.abort();
    }, [report_id, error]);

    return { data, loading, error };
};
