import FlagIcon from '@mui/icons-material/Flag';
import Button from '@mui/material/Button';
import Typography from '@mui/material/Typography';
import { useNavigate } from '@tanstack/react-router';
import {
    createColumnHelper,
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
import { TableHeadingCell } from './TableHeadingCell.tsx';

const columnHelper = createColumnHelper<PersonMessage>();

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

    const columns = [
        columnHelper.accessor('server_id', {
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
                                search: (prev) => ({ ...prev, server_id: info.getValue() })
                            });
                        }}
                    >
                        {messages[info.row.index].server_name}
                    </Button>
                );
            }
        }),
        columnHelper.accessor('created_on', {
            header: () => <TableHeadingCell name={'Created'} />,
            cell: (info) => <Typography align={'center'}>{renderDateTime(info.getValue())}</Typography>
        }),
        columnHelper.accessor('persona_name', {
            header: () => <TableHeadingCell name={'Name'} />,
            cell: (info) => (
                <PersonCell
                    steam_id={messages[info.row.index].steam_id}
                    avatar_hash={messages[info.row.index].avatar_hash}
                    personaname={messages[info.row.index].persona_name}
                    onClick={async () => {
                        await navigate({
                            params: { steamId: messages[info.row.index].steam_id },
                            to: `/profile/$steamId`
                        });
                    }}
                />
            )
        }),
        columnHelper.accessor('body', {
            header: () => <TableHeadingCell name={'Message'} />,
            cell: (info) => (
                <Typography padding={0} variant={'body1'}>
                    {info.getValue()}
                </Typography>
            )
        }),
        columnHelper.accessor('auto_filter_flagged', {
            header: () => <TableHeadingCell name={'F'} />,
            cell: (info) => (info.getValue() > 0 ? <FlagIcon color={'error'} /> : <></>)
        })
    ];
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
