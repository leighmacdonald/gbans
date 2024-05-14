import { useMemo } from 'react';
import FlagIcon from '@mui/icons-material/Flag';
import ReportIcon from '@mui/icons-material/Report';
import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { useNavigate } from '@tanstack/react-router';
import {
    ColumnDef,
    getCoreRowModel,
    getPaginationRowModel,
    OnChangeFn,
    PaginationState,
    TableOptions,
    useReactTable
} from '@tanstack/react-table';
import stc from 'string-to-color';
import { PersonMessage } from '../api';
import { renderDateTime } from '../util/text.tsx';
import { DataTable } from './DataTable.tsx';
import { PersonCell } from './PersonCell.tsx';
import RouterLink from './RouterLink.tsx';
import { TableHeadingCell } from './TableHeadingCell.tsx';

export const ChatTable = ({
    messages,
    isLoading,
    manualPaging = true,
    pagination,
    setPagination
}: {
    messages: PersonMessage[];
    isLoading: boolean;
    manualPaging?: boolean;
    pagination?: PaginationState;
    setPagination?: OnChangeFn<PaginationState>;
}) => {
    const navigate = useNavigate({ from: '/chatlogs' });
    const columns = useMemo<ColumnDef<PersonMessage>[]>(
        () => [
            {
                accessorKey: 'server_id',
                header: () => <TableHeadingCell name={'Server'} />,
                cell: (info) => {
                    return (
                        <Button
                            sx={{
                                color: stc(messages[info.row.index].server_name)
                            }}
                            onClick={async () => {
                                await navigate({
                                    to: '/chatlogs',
                                    search: (prev) => ({ ...prev, server_id: info.getValue() as number })
                                });
                            }}
                        >
                            {messages[info.row.index].server_name}
                        </Button>
                    );
                }
            },
            {
                accessorKey: 'created_on',
                header: () => <TableHeadingCell name={'Created'} />,
                cell: (info) => <Typography align={'center'}>{renderDateTime(info.getValue() as Date)}</Typography>
            },
            {
                accessorKey: 'persona_name',
                header: () => <TableHeadingCell name={'Name'} />,
                cell: (info) => (
                    <PersonCell
                        showCopy={true}
                        steam_id={messages[info.row.index].steam_id}
                        avatar_hash={messages[info.row.index].avatar_hash}
                        personaname={messages[info.row.index].persona_name}
                    />
                )
            },
            {
                accessorKey: 'body',
                header: () => <TableHeadingCell name={'Message'} />,
                cell: (info) => (
                    <Typography padding={0} variant={'body1'}>
                        {info.getValue() as string}
                    </Typography>
                )
            },
            {
                accessorKey: 'auto_filter_flagged',
                header: () => <TableHeadingCell name={''} />,
                cell: (info) =>
                    (info.getValue() as number) > 0 ? (
                        <Tooltip title={'Message already flagged'}>
                            <FlagIcon color={'error'} />
                        </Tooltip>
                    ) : (
                        <></>
                    )
            },
            {
                id: 'actions',
                header: () => <TableHeadingCell name={''} />,
                cell: (info) => (
                    <Tooltip title={'Create Report'}>
                        <IconButton
                            color={'error'}
                            disabled={info.row.original.auto_filter_flagged > 0}
                            component={RouterLink}
                            to={'/report'}
                            search={{
                                person_message_id: info.row.original.person_message_id,
                                steam_id: info.row.original.steam_id
                            }}
                        >
                            <ReportIcon />
                        </IconButton>
                    </Tooltip>
                )
            }
        ],
        [messages, navigate]
    );

    const opts: TableOptions<PersonMessage> = {
        data: messages,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        manualPagination: manualPaging,
        autoResetPageIndex: true,
        ...(manualPaging
            ? {}
            : {
                  manualPagination: false,
                  onPaginationChange: setPagination,
                  getPaginationRowModel: getPaginationRowModel(),
                  state: { pagination }
              })
    };

    const table = useReactTable(opts);

    return <DataTable table={table} isLoading={isLoading} />;
};
