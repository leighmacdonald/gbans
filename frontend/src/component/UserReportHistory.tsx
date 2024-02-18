import { ChangeEvent, useState } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import VisibilityIcon from '@mui/icons-material/Visibility';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { ReportStatus, reportStatusString, ReportWithAuthor } from '../api';
import { useReports } from '../hooks/useReports';
import { Order, RowsPerPage } from '../util/table.ts';
import { PersonCell } from './PersonCell';
import { ReportStatusIcon } from './ReportStatusIcon';
import { LazyTable } from './table/LazyTable';

export const UserReportHistory = ({ steam_id }: { steam_id: string }) => {
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] =
        useState<keyof ReportWithAuthor>('created_on');
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        RowsPerPage.TwentyFive
    );
    const [page, setPage] = useState(0);

    const { data, count } = useReports({
        order_by: 'created_on',
        source_id: steam_id,
        deleted: false,
        offset: page * rowPerPageCount,
        limit: rowPerPageCount,
        desc: sortOrder == 'desc',
        report_status: ReportStatus.Any
    });

    return (
        <LazyTable
            showPager={true}
            count={count}
            rows={data}
            page={page}
            rowsPerPage={rowPerPageCount}
            sortOrder={sortOrder}
            sortColumn={sortColumn}
            onSortColumnChanged={async (column) => {
                setSortColumn(column);
            }}
            onSortOrderChanged={async (direction) => {
                setSortOrder(direction);
            }}
            onPageChange={(_, newPage: number) => {
                setPage(newPage);
            }}
            onRowsPerPageChange={(
                event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>
            ) => {
                setRowPerPageCount(parseInt(event.target.value, 10));
                setPage(0);
            }}
            columns={[
                {
                    label: 'Status',
                    tooltip: 'Report Status',
                    sortKey: 'report_status',
                    sortable: true,
                    align: 'left',
                    renderer: (obj) => (
                        <Stack direction={'row'} spacing={1}>
                            <ReportStatusIcon
                                reportStatus={obj.report_status}
                            />
                            <Typography variant={'body1'}>
                                {reportStatusString(obj.report_status)}
                            </Typography>
                        </Stack>
                    )
                },
                {
                    label: 'Player',
                    tooltip: 'Reported Player',
                    sortKey: 'subject',
                    sortable: true,
                    align: 'left',
                    renderer: (row) => (
                        <PersonCell
                            steam_id={row.subject.steam_id}
                            personaname={row.subject.personaname}
                            avatar_hash={row.subject.avatarhash}
                        />
                    )
                },
                {
                    label: 'View',
                    tooltip: 'View your report',
                    sortable: false,
                    virtual: true,
                    virtualKey: 'actions',
                    align: 'right',
                    renderer: (row) => (
                        <ButtonGroup>
                            <IconButton
                                color={'primary'}
                                component={RouterLink}
                                to={`/report/${row.report_id}`}
                            >
                                <Tooltip title={'View'}>
                                    <VisibilityIcon />
                                </Tooltip>
                            </IconButton>
                        </ButtonGroup>
                    )
                }
            ]}
        />
    );
};
