import { TableFooter, TableSortLabel } from '@mui/material';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Typography from '@mui/material/Typography';
import { ColumnSort, flexRender, Table as TSTable } from '@tanstack/react-table';
import { LoadingPlaceholder } from './LoadingPlaceholder.tsx';
import { TableCellSmall } from './TableCellSmall.tsx';

export const DataTable = <T,>({
    table,
    isLoading,
    onSort,
    padding = 'none'
}: {
    table: TSTable<T>;
    isLoading: boolean;
    padding?: 'normal' | 'checkbox' | 'none';
    onSort?: (sortColumn: ColumnSort) => void;
}) => {
    if (isLoading) {
        return <LoadingPlaceholder />;
    }

    return (
        <TableContainer>
            <Table padding={padding}>
                <TableHead>
                    {table.getHeaderGroups().map((headerGroup) => (
                        <TableRow key={headerGroup.id}>
                            {headerGroup.headers.map((header) => {
                                const direction = header.column.getIsSorted() ? header.column.getIsSorted() : undefined;
                                return (
                                    <TableCellSmall key={header.id}>
                                        {header.column.getCanSort() ? (
                                            <TableSortLabel
                                                active={Boolean(header.column.getIsSorted())}
                                                direction={direction ? direction : undefined}
                                                onClick={(event) => {
                                                    const fn = header.column.getToggleSortingHandler();
                                                    onSort && onSort({ id: header.id, desc: direction == 'desc' });
                                                    fn && fn(event);
                                                }}
                                            >
                                                <Typography
                                                    padding={0}
                                                    sx={{
                                                        fontWeight: 'bold'
                                                    }}
                                                    variant={'button'}
                                                >
                                                    {header.isPlaceholder
                                                        ? null
                                                        : flexRender(
                                                              header.column.columnDef.header,
                                                              header.getContext()
                                                          )}
                                                </Typography>
                                            </TableSortLabel>
                                        ) : (
                                            <Typography
                                                padding={0}
                                                sx={{
                                                    fontWeight: 'bold'
                                                }}
                                                variant={'button'}
                                            >
                                                {header.isPlaceholder
                                                    ? null
                                                    : flexRender(header.column.columnDef.header, header.getContext())}
                                            </Typography>
                                        )}
                                    </TableCellSmall>
                                );
                            })}
                        </TableRow>
                    ))}
                </TableHead>
                <TableBody>
                    {table.getRowModel().rows.map((row) => (
                        <TableRow key={row.id} hover>
                            {row.getVisibleCells().map((cell) => (
                                <TableCell key={cell.id}>
                                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                                </TableCell>
                            ))}
                        </TableRow>
                    ))}
                </TableBody>
                <TableFooter>
                    {table.getFooterGroups().map((footerGroup) => (
                        <TableRow key={footerGroup.id}>
                            {footerGroup.headers.map((header) => (
                                <TableCell key={header.id}>
                                    {header.isPlaceholder
                                        ? null
                                        : flexRender(header.column.columnDef.footer, header.getContext())}
                                </TableCell>
                            ))}
                        </TableRow>
                    ))}
                </TableFooter>
            </Table>
        </TableContainer>
    );
};
