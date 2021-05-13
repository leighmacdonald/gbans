import {
    Checkbox,
    createStyles,
    FormControlLabel,
    IconButton,
    lighten,
    Paper,
    Switch,
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TablePagination,
    TableRow,
    TableSortLabel,
    Toolbar,
    Tooltip,
    Typography
} from '@material-ui/core';
import React, { useEffect, useState } from 'react';
import { makeStyles, Theme } from '@material-ui/core/styles';
import clsx from 'clsx';
import DeleteIcon from '@material-ui/icons/Delete';
import FilterListIcon from '@material-ui/icons/FilterList';
import { StyledTableCell } from './Tables';

function descendingComparator<T>(a: T, b: T, orderBy: keyof T) {
    if (b[orderBy] < a[orderBy]) {
        return -1;
    }
    if (b[orderBy] > a[orderBy]) {
        return 1;
    }
    return 0;
}

export type Order = 'asc' | 'desc';

const stableSort = <T extends unknown>(
    array: T[],
    comparator: (a: T, b: T) => number
) => {
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
    numeric: boolean;
}

interface EnhancedTableProps<TRecord> {
    classes: ReturnType<typeof useStyles>;
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

const useToolbarStyles = makeStyles((theme: Theme) =>
    createStyles({
        root: {
            paddingLeft: theme.spacing(2),
            paddingRight: theme.spacing(1)
        },
        highlight:
            theme.palette.type === 'light'
                ? {
                      color: theme.palette.secondary.main,
                      backgroundColor: lighten(
                          theme.palette.secondary.light,
                          0.85
                      )
                  }
                : {
                      color: theme.palette.text.primary,
                      backgroundColor: theme.palette.secondary.dark
                  },
        title: {
            flex: '1 1 100%'
        }
    })
);

export interface EnhancedTableToolbarProps {
    numSelected: number;
    heading: string;
}

export const useStyles = makeStyles((theme: Theme) =>
    createStyles({
        root: {
            width: '100%'
        },
        paper: {
            width: '100%',
            marginBottom: theme.spacing(2)
        },
        table: {
            minWidth: 750
        },
        visuallyHidden: {
            border: 0,
            clip: 'rect(0 0 0 0)',
            height: 1,
            margin: -1,
            overflow: 'hidden',
            padding: 0,
            position: 'absolute',
            top: 20,
            width: 1
        }
    })
);

export interface TableProps<TRecord> {
    headers: HeadCell<TRecord>[];
    heading: string;
    id_field: keyof TRecord;
    connector: () => Promise<TRecord[]>;
    showToolbar: boolean;
}

export const CreateDataTable = <TRecord extends unknown>(): ((
    props: TableProps<TRecord>
) => JSX.Element) => {
    const classes = useToolbarStyles();
    const EnhancedTableToolbar = (props: EnhancedTableToolbarProps) => {
        const { numSelected, heading } = props;

        return (
            <Toolbar
                className={clsx(classes.root, {
                    [classes.highlight]: numSelected > 0
                })}
            >
                {numSelected > 0 ? (
                    <Typography
                        className={classes.title}
                        variant="subtitle1"
                        component="div"
                    >
                        {numSelected} selected
                    </Typography>
                ) : (
                    <Typography
                        className={classes.title}
                        variant="h6"
                        id="tableTitle"
                        component="div"
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
            classes,
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
                    <StyledTableCell padding="checkbox">
                        <Checkbox
                            indeterminate={
                                numSelected > 0 && numSelected < rowCount
                            }
                            checked={rowCount > 0 && numSelected === rowCount}
                            onChange={onSelectAllClick}
                            inputProps={{ 'aria-label': 'select all desserts' }}
                        />
                    </StyledTableCell>
                    {props.headers.map((headCell, index) => (
                        <StyledTableCell
                            key={`${headCell.id}-${index}`}
                            align={headCell.numeric ? 'right' : 'left'}
                            padding={
                                headCell.disablePadding ? 'none' : 'default'
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
                                    <span className={classes.visuallyHidden}>
                                        {order === 'desc'
                                            ? 'sorted descending'
                                            : 'sorted ascending'}
                                    </span>
                                ) : null}
                            </TableSortLabel>
                        </StyledTableCell>
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
        const classes = useStyles();
        // eslint-disable-next-line
        const [order, setOrder] = React.useState<Order>('asc');
        // eslint-disable-next-line
        const [orderBy, setOrderBy] = React.useState<keyof TRecord>(id_field);
        // eslint-disable-next-line
        const [selected, setSelected] = React.useState<string[]>([]);
        // eslint-disable-next-line
        const [page, setPage] = React.useState(0);
        // eslint-disable-next-line
        const [dense, setDense] = React.useState(false);
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

        const handleChangeDense = (
            event: React.ChangeEvent<HTMLInputElement>
        ) => {
            setDense(event.target.checked);
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
            <div className={classes.root}>
                <Paper className={classes.paper}>
                    {showToolbar && (
                        <EnhancedTableToolbar
                            numSelected={selected.length}
                            heading={heading}
                        />
                    )}
                    <TableContainer>
                        <Table
                            className={classes.table}
                            aria-labelledby="tableTitle"
                            size={dense ? 'small' : 'medium'}
                            aria-label="enhanced table"
                        >
                            <EnhancedTableHead
                                classes={classes}
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
                                            height:
                                                (dense ? 33 : 53) * emptyRows
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
                        onChangePage={handleChangePage}
                        onChangeRowsPerPage={handleChangeRowsPerPage}
                    />
                </Paper>
                <FormControlLabel
                    control={
                        <Switch checked={dense} onChange={handleChangeDense} />
                    }
                    label="Dense padding"
                />
            </div>
        );
    };
};
