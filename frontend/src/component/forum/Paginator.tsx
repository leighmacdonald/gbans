import { TablePagination } from '@mui/material';
import { useNavigate } from '@tanstack/react-router';
import { LazyResult } from '../../util/table.ts';

export const Paginator = <T,>({
    data,
    page,
    rows,
    path
}: {
    data?: LazyResult<T>;
    page: number;
    rows: number;
    path: string;
}) => {
    const navigate = useNavigate();

    return (
        <TablePagination
            component={'div'}
            count={data ? data.count : 0}
            page={page}
            rowsPerPage={rows}
            onRowsPerPageChange={async (event) => {
                await navigate({ to: path, search: (search) => ({ ...search, pageSize: Number(event.target.value) }) });
            }}
            onPageChange={async (_, newPage: number) => {
                await navigate({ to: path, search: (search) => ({ ...search, pageIndex: newPage }) });
            }}
        />
    );
};
