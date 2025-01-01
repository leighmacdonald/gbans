import { useMemo } from 'react';
import FlagIcon from '@mui/icons-material/Flag';
import ReportIcon from '@mui/icons-material/Report';
import Button from '@mui/material/Button';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
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
import { PersonMessage } from '../api';
import { stringToColour } from '../util/colours.ts';
import { renderDateTime } from '../util/time.ts';
import { DataTable } from './DataTable.tsx';
import { IconButtonLink } from './IconButtonLink.tsx';
import { PersonCell } from './PersonCell.tsx';

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
    const theme = useTheme();

    const columns = useMemo<ColumnDef<PersonMessage>[]>(
        () => [
            {
                accessorKey: 'server_id',
                header: 'Server',
                size: 40,
                cell: (info) => {
                    return (
                        <Button
                            sx={{
                                color: stringToColour(messages[info.row.index].server_name, theme.palette.mode)
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
                header: 'Created',
                size: 80,
                cell: (info) => <Typography align={'center'}>{renderDateTime(info.getValue() as Date)}</Typography>
            },
            {
                accessorKey: 'persona_name',
                header: 'Name',
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
                header: 'Message',
                size: 400,
                cell: (info) => (
                    <Typography padding={0} variant={'body1'}>
                        {info.getValue() as string}
                    </Typography>
                )
            },
            {
                accessorKey: 'auto_filter_flagged',
                size: 30,
                header: '',
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
                size: 30,
                cell: (info) => (
                    <Tooltip title={'Create Report'}>
                        <IconButtonLink
                            color={'error'}
                            disabled={info.row.original.auto_filter_flagged > 0}
                            to={'/report'}
                            search={{
                                person_message_id: info.row.original.person_message_id,
                                steam_id: info.row.original.steam_id
                            }}
                        >
                            <ReportIcon />
                        </IconButtonLink>
                    </Tooltip>
                )
            }
        ],
        [messages, navigate, theme.palette.mode]
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
