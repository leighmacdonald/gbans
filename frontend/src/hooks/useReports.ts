import { useEffect, useState } from 'react';
import { apiGetReports, ReportQueryFilter, ReportWithAuthor } from '../api';
import { logErr } from '../util/errors';
import { hookResult } from './hookResult';

export const useReports = (
    opts: ReportQueryFilter
): hookResult<ReportWithAuthor[]> => {
    const [loading, setLoading] = useState(false);
    const [count, setCount] = useState<number>(0);
    const [data, setData] = useState<ReportWithAuthor[]>([]);

    useEffect(() => {
        setLoading(true);
        apiGetReports({
            limit: opts.limit,
            offset: opts.offset,
            order_by: opts.order_by,
            desc: opts.desc,
            report_status: opts.report_status,
            source_id: opts.source_id,
            target_id: opts.target_id
        })
            .then((resp) => {
                setData(resp.data || []);
                setCount(resp.count);
            })
            .catch((e) => {
                logErr(e);
            })
            .finally(() => {
                setLoading(false);
            });
    }, [
        opts.desc,
        opts.limit,
        opts.offset,
        opts.order_by,
        opts.report_status,
        opts.source_id,
        opts.target_id
    ]);

    return { data, count, loading };
};
