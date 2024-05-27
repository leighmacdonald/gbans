import { useMemo } from 'react';
import ChevronRightIcon from '@mui/icons-material/ChevronRight';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { createColumnHelper, getCoreRowModel, TableOptions, useReactTable } from '@tanstack/react-table';
import { BaseServer, cleanMapName } from '../api';
import { useMapStateCtx } from '../hooks/useMapStateCtx.ts';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { tf2Fonts } from '../theme';
import { logErr } from '../util/errors';
import { DataTable } from './DataTable.tsx';
import { Flag } from './Flag';
import { LoadingSpinner } from './LoadingSpinner';
import { TableHeadingCell } from './TableHeadingCell.tsx';

type ServerRow = BaseServer & { copy: string; connect: string };

export const ServerList = () => {
    const { sendFlash } = useUserFlashCtx();
    const { selectedServers } = useMapStateCtx();

    const columnHelper = createColumnHelper<ServerRow>();

    const columns = [
        columnHelper.accessor('cc', {
            enableSorting: false,
            header: () => <TableHeadingCell name={'CC'} />,
            cell: (info) => <Flag countryCode={info.getValue()} />
        }),
        columnHelper.accessor('name', {
            enableSorting: false,
            header: () => <TableHeadingCell name={'Server'} />,
            cell: (info) => (
                <Typography variant={'button'} fontFamily={tf2Fonts}>
                    {info.getValue()}
                </Typography>
            )
        }),
        columnHelper.accessor('map', {
            enableSorting: false,
            header: () => <TableHeadingCell name={'Map'} />,
            cell: (info) => <Typography variant={'body2'}>{cleanMapName(info.getValue())}</Typography>
        }),
        columnHelper.accessor('players', {
            enableSorting: false,
            header: () => <TableHeadingCell name={'Players'} />,
            cell: (info) => (
                <Typography
                    variant={'body2'}
                >{`${info.getValue()}/${selectedServers[info.row.index].max_players}`}</Typography>
            )
        }),
        columnHelper.accessor('distance', {
            enableSorting: false,
            header: () => <TableHeadingCell name={'Dist'} />,
            cell: (info) => (
                <Tooltip title={`Distance in hammer units: ${Math.round((info.getValue() ?? 1) * 52.49)} khu`}>
                    <Typography variant={'caption'}>{`${info.getValue().toFixed(0)}km`}</Typography>
                </Tooltip>
            )
        }),
        columnHelper.accessor('copy', {
            enableSorting: false,
            header: () => <TableHeadingCell name={'Cp'} />,
            cell: (info) => (
                <IconButton
                    color={'primary'}
                    aria-label={'Copy connect string to clipboard'}
                    onClick={() => {
                        navigator.clipboard
                            .writeText(
                                `connect ${selectedServers[info.row.index].host}:${selectedServers[info.row.index].port}`
                            )
                            .then(() => {
                                sendFlash('success', 'Copied address to clipboard');
                            })
                            .catch((e) => {
                                sendFlash('error', 'Failed to copy address');
                                logErr(e);
                            });
                    }}
                >
                    <ContentCopyIcon />
                </IconButton>
            )
        }),
        columnHelper.accessor('connect', {
            enableSorting: false,
            header: () => <TableHeadingCell name={'Connect'} />,
            cell: (info) => (
                <Button
                    fullWidth
                    endIcon={<ChevronRightIcon />}
                    component={Link}
                    href={`steam://connect/${selectedServers[info.row.index].ip}:${selectedServers[info.row.index].port}`}
                    variant={'contained'}
                    sx={{ minWidth: 100 }}
                >
                    Join
                </Button>
            )
        })
    ];

    const metaServers = useMemo(() => {
        return selectedServers.map((s) => ({ ...s, copy: '', connect: '' }));
    }, [selectedServers]);

    const opts: TableOptions<ServerRow> = {
        data: metaServers,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        manualPagination: false,
        autoResetPageIndex: true
    };

    const table = useReactTable(opts);

    if (selectedServers.length === 0) {
        return <LoadingSpinner />;
    }

    return <DataTable table={table} isLoading={false} />;
};
