import { TablePagination } from '@mui/material';
import { useNavigate } from '@tanstack/react-router';
import { LazyResult } from '../util/table.ts';

export const Paginator = <T,>({ data, page, rows }: { data?: LazyResult<T>; page: number; rows: number }) => {
    const navigate = useNavigate();

    return (
        <TablePagination
            count={data ? data.count : 0}
            page={page}
            rowsPerPage={rows}
            onRowsPerPageChange={async (event) => {
                await navigate({ search: (search) => ({ ...search, rows: Number(event.target.value) }) });
            }}
            onPageChange={async (_, newPage: number) => {
                await navigate({ search: (search) => ({ ...search, page: newPage }) });
            }}
        />
    );
};
