import React, { useEffect, useState } from 'react';
import Toolbar from '@mui/material/Toolbar';
import {
    Checkbox,
    IconButton,
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TablePagination,
    TableRow,
    TableSortLabel,
    Tooltip,
    Typography
} from '@mui/material';
import Paper from '@mui/material/Paper';
import DeleteIcon from '@mui/icons-material/Delete';
import FilterListIcon from '@mui/icons-material/FilterList';

const descendingComparator = <T,>(a: T, b: T, orderBy: keyof T) => {
    if (b[orderBy] < a[orderBy]) {
        return -1;
    }
    if (b[orderBy] > a[orderBy]) {
        return 1;
    }
    return 0;
};

export type Order = 'asc' | 'desc';

const stableSort = <T,>(array: T[], comparator: (a: T, b: T) => number) => {
    const stabilizedThis = array.map((el, index) => [el, index] as [T, number]);
    stabilizedThis.sort((a, b) => {
        const order = comparator(a[0], b[0]);
        if (order !== 0) return order;
        return a[1] - b[1];
    });
    return stabilizedThis.map((el) => el[0]);
};

interface HeadCell<TRecord> {
    disablePadding: boolean;
    id: keyof TRecord;
    label: string;
    cell_type: 'steam_id' | 'number' | 'bool' | 'string' | 'date' | 'flag';
}

interface EnhancedTableProps<TRecord> {
    classes?: string[];
    numSelected: number;
    onRequestSort: (
        event: React.MouseEvent<unknown>,
        property: keyof TRecord
    ) => void;
    onSelectAllClick: (event: React.ChangeEvent<HTMLInputElement>) => void;
    order: Order;
    orderBy: keyof TRecord;
    rowCount: number;
    headers: HeadCell<TRecord>[];
}

export interface EnhancedTableToolbarProps {
    numSelected: number;
    heading: string;
}

export interface TableProps<TRecord> {
    headers: HeadCell<TRecord>[];
    heading: string;
    id_field: keyof TRecord;
    connector: () => Promise<TRecord[]>;
    showToolbar: boolean;
}

export const CreateDataTable = <TRecord,>(): ((
    props: TableProps<TRecord>
) => JSX.Element) => {
    const EnhancedTableToolbar = (props: EnhancedTableToolbarProps) => {
        const { numSelected, heading } = props;

        return (
            <Toolbar
            // className={clsx(classes.root, {[classes.highlight]: numSelected > 0})}
            >
                {numSelected > 0 ? (
                    <Typography variant={'subtitle1'} component={'div'}>
                        {numSelected} selected
                    </Typography>
                ) : (
                    <Typography
                        variant={'h6'}
                        id={'tableTitle'}
                        component={'div'}
                    >
                        {heading}
                    </Typography>
                )}
                {numSelected > 0 ? (
                    <Tooltip title="Delete">
                        <IconButton aria-label="delete">
                            <DeleteIcon />
                        </IconButton>
                    </Tooltip>
                ) : (
                    <Tooltip title="Filter list">
                        <IconButton aria-label="filter list">
                            <FilterListIcon />
                        </IconButton>
                    </Tooltip>
                )}
            </Toolbar>
        );
    };

    function EnhancedTableHead<TRecord>(props: EnhancedTableProps<TRecord>) {
        const {
            onSelectAllClick,
            order,
            orderBy,
            numSelected,
            rowCount,
            onRequestSort
        } = props;
        const createSortHandler =
            (property: keyof TRecord) => (event: React.MouseEvent<unknown>) => {
                onRequestSort(event, property);
            };

        return (
            <TableHead>
                <TableRow>
                    <TableCell padding="checkbox">
                        <Checkbox
                            indeterminate={
                                numSelected > 0 && numSelected < rowCount
                            }
                            checked={rowCount > 0 && numSelected === rowCount}
                            onChange={onSelectAllClick}
                            inputProps={{ 'aria-label': 'select all desserts' }}
                        />
                    </TableCell>
                    {props.headers.map((headCell, index) => (
                        <TableCell
                            key={`${headCell.id}-${index}`}
                            align={
                                headCell.cell_type === 'number'
                                    ? 'right'
                                    : 'left'
                            }
                            padding={
                                headCell.disablePadding ? 'none' : 'normal'
                            }
                            sortDirection={
                                orderBy === headCell.id ? order : false
                            }
                        >
                            <TableSortLabel
                                active={orderBy === headCell.id}
                                direction={
                                    orderBy === headCell.id ? order : 'asc'
                                }
                                onClick={createSortHandler(headCell.id)}
                            >
                                {headCell.label}
                                {orderBy === headCell.id ? (
                                    <span>
                                        {order === 'desc'
                                            ? 'sorted descending'
                                            : 'sorted ascending'}
                                    </span>
                                ) : null}
                            </TableSortLabel>
                        </TableCell>
                    ))}
                </TableRow>
            </TableHead>
        );
    }

    // eslint-disable-next-line react/display-name
    return ({
        connector,
        id_field,
        headers,
        heading,
        showToolbar
    }: TableProps<TRecord>) => {
        // eslint-disable-next-line
        const [rows, setRows] = useState<TRecord[]>([]);
        // eslint-disable-next-line
        useEffect(() => {
            const loadData = async () => {
                const resp = (await connector()) as TRecord[];
                setRows(resp ?? []);
            };
            // noinspection JSIgnoredPromiseFromCall
            loadData();
            // eslint-disable-next-line react-hooks/exhaustive-deps
        }, []);

        // eslint-disable-next-line
        const [order, setOrder] = React.useState<Order>('asc');
        // eslint-disable-next-line
        const [orderBy, setOrderBy] = React.useState<keyof TRecord>(id_field);
        // eslint-disable-next-line
        const [selected, setSelected] = React.useState<string[]>([]);
        // eslint-disable-next-line
        const [page, setPage] = React.useState(0);
        // eslint-disable-next-line
        const [rowsPerPage, setRowsPerPage] = React.useState(10);
        // eslint-disable-next-line

        const handleRequestSort = (
            _: React.MouseEvent<unknown>,
            property: keyof TRecord
        ) => {
            const isAsc = orderBy === property && order === 'asc';
            setOrder(isAsc ? 'desc' : 'asc');
            setOrderBy(property);
        };

        const handleSelectAllClick = (
            event: React.ChangeEvent<HTMLInputElement>
        ) => {
            if (event.target.checked) {
                const newSelected = rows.map((n) => `${n[id_field]}`);
                setSelected(newSelected);
                return;
            }
            setSelected([]);
        };

        const handleClick = (_: React.MouseEvent<unknown>, name: string) => {
            const selectedIndex = selected.indexOf(name);
            let newSelected: string[] = [];

            if (selectedIndex === -1) {
                newSelected = newSelected.concat(selected, name);
            } else if (selectedIndex === 0) {
                newSelected = newSelected.concat(selected.slice(1));
            } else if (selectedIndex === selected.length - 1) {
                newSelected = newSelected.concat(selected.slice(0, -1));
            } else if (selectedIndex > 0) {
                newSelected = newSelected.concat(
                    selected.slice(0, selectedIndex),
                    selected.slice(selectedIndex + 1)
                );
            }

            setSelected(newSelected);
        };

        const handleChangePage = (_: unknown, newPage: number) => {
            setPage(newPage);
        };

        const handleChangeRowsPerPage = (
            event: React.ChangeEvent<HTMLInputElement>
        ) => {
            setRowsPerPage(parseInt(event.target.value, 10));
            setPage(0);
        };

        const isSelected = (name: string) => selected.indexOf(name) !== -1;

        const emptyRows =
            rowsPerPage -
            Math.min(rowsPerPage, rows.length - page * rowsPerPage);
        const c = function (
            order: Order,
            orderBy: keyof TRecord
        ): (a: TRecord, b: TRecord) => number {
            return order === 'desc'
                ? (a, b) => descendingComparator(a, b, orderBy)
                : (a, b) => -descendingComparator(a, b, orderBy);
        };
        return (
            <div>
                <Paper>
                    {showToolbar && (
                        <EnhancedTableToolbar
                            numSelected={selected.length}
                            heading={heading}
                        />
                    )}
                    <TableContainer>
                        <Table
                            aria-labelledby="tableTitle"
                            size={'small'}
                            aria-label="enhanced table"
                        >
                            <EnhancedTableHead
                                numSelected={selected.length}
                                order={order}
                                orderBy={orderBy}
                                onSelectAllClick={handleSelectAllClick}
                                onRequestSort={handleRequestSort}
                                rowCount={rows.length}
                                headers={headers}
                            />
                            <TableBody>
                                {stableSort<TRecord>(rows, c(order, orderBy))
                                    .slice(
                                        page * rowsPerPage,
                                        page * rowsPerPage + rowsPerPage
                                    )
                                    .map((row, index) => {
                                        const isItemSelected = isSelected(
                                            `${row[id_field]}-${index}`
                                        );
                                        const labelId = `enhanced-table-checkbox-${index}`;

                                        return (
                                            <TableRow
                                                hover
                                                onClick={(event) =>
                                                    handleClick(
                                                        event,
                                                        `${row[id_field]}-${index}`
                                                    )
                                                }
                                                role="checkbox"
                                                aria-checked={isItemSelected}
                                                tabIndex={-1}
                                                key={`${row[id_field]}-${index}`}
                                                selected={isItemSelected}
                                            >
                                                <TableCell padding="checkbox">
                                                    <Checkbox
                                                        checked={isItemSelected}
                                                        inputProps={{
                                                            'aria-labelledby':
                                                                labelId
                                                        }}
                                                    />
                                                </TableCell>
                                                {headers.map((h, i) => {
                                                    if (i === 0) {
                                                        return (
                                                            <TableCell
                                                                key={`cell-${index}-${i}`}
                                                                component="th"
                                                                id={labelId}
                                                                scope="row"
                                                                padding="none"
                                                            >
                                                                {row[h.id]}
                                                            </TableCell>
                                                        );
                                                    }
                                                    return (
                                                        <TableCell
                                                            key={`cell-${index}-${i}`}
                                                            align="right"
                                                        >
                                                            {row[h.id]}
                                                        </TableCell>
                                                    );
                                                })}
                                            </TableRow>
                                        );
                                    })}
                                {emptyRows > 0 && (
                                    <TableRow
                                        style={{
                                            height: 33 * emptyRows
                                        }}
                                    >
                                        <TableCell colSpan={6} />
                                    </TableRow>
                                )}
                            </TableBody>
                        </Table>
                    </TableContainer>
                    <TablePagination
                        rowsPerPageOptions={[5, 10, 25]}
                        component="div"
                        count={rows.length}
                        rowsPerPage={rowsPerPage}
                        page={page}
                        onPageChange={handleChangePage}
                        onRowsPerPageChange={handleChangeRowsPerPage}
                    />
                </Paper>
            </div>
        );
    };
};
