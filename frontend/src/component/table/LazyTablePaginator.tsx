import { MouseEvent, ChangeEvent } from 'react';
import { TablePagination } from '@mui/material';
import { RowsPerPage } from './LazyTable';

interface LazyTablePaginatorProps {
    loading: boolean;
    page: number;
    total: number;
    rowsPerPage: RowsPerPage;
    onRowsPerPageChange: (event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => void;
    onPageChange: (event: MouseEvent<HTMLButtonElement> | null, page: number) => void;
}

export const LazyTablePaginator = ({ loading, page, total, rowsPerPage, onPageChange, onRowsPerPageChange }: LazyTablePaginatorProps) => {
    return (
        <TablePagination
            SelectProps={{
                disabled: loading
            }}
            backIconButtonProps={
                loading
                    ? {
                          disabled: loading
                      }
                    : undefined
            }
            nextIconButtonProps={
                loading
                    ? {
                          disabled: loading
                      }
                    : undefined
            }
            variant={'head'}
            page={page}
            count={total}
            showFirstButton
            showLastButton
            rowsPerPage={rowsPerPage}
            onPageChange={onPageChange}
            onRowsPerPageChange={onRowsPerPageChange}
        />
    );
};
