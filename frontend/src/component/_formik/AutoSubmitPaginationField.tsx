import { useEffect } from 'react';
import { useFormikContext } from 'formik';
import { logErr } from '../../util/errors';
import { RowsPerPage } from '../table/LazyTable';

export const AutoSubmitPaginationField = ({ page, rowsPerPage }: { page: number; rowsPerPage: RowsPerPage }) => {
    const { submitForm } = useFormikContext();

    useEffect(() => {
        submitForm().catch(logErr);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [page, rowsPerPage]);

    return <></>;
};
