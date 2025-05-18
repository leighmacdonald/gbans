import { useMemo } from 'react';
import ChevronRightIcon from '@mui/icons-material/ChevronRight';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import GroupsIcon from '@mui/icons-material/Groups';
import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { ColumnDef, createColumnHelper, getCoreRowModel, TableOptions, useReactTable } from '@tanstack/react-table';
import { z } from 'zod';
import { cleanMapName, PermissionLevel } from '../api';
import { useAuth } from '../hooks/useAuth.ts';
import { useMapStateCtx } from '../hooks/useMapStateCtx.ts';
import { useQueueCtx } from '../hooks/useQueueCtx.ts';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { schemaServerRow } from '../schema/server.ts';
import { tf2Fonts } from '../theme';
import { logErr } from '../util/errors';
import { Flag } from './Flag';
import { StyledBadge } from './StyledBadge.tsx';
import { DataTable } from './table/DataTable.tsx';

type ServerRow = z.infer<typeof schemaServerRow>;

export const ServerList = () => {
    const { sendFlash } = useUserFlashCtx();
    const { profile, hasPermission } = useAuth();
    const { selectedServers } = useMapStateCtx();
    const { joinQueue, leaveQueue, lobbies } = useQueueCtx();
    const columnHelper = createColumnHelper<ServerRow>();

    const metaServers = useMemo(() => {
        return selectedServers.map((s) => ({ ...s, copy: '', connect: '' }));
    }, [selectedServers]);

    const isQueued = (server_id: number) => {
        try {
            return Boolean(
                lobbies.find((s) => s.server_id == server_id)?.members?.find((m) => m.steam_id == profile.steam_id)
            );
        } catch {
            return false;
        }
    };

    const columns = useMemo(() => {
        return [
            columnHelper.accessor('cc', {
                header: 'CC',
                size: 40,
                cell: (info) => <Flag countryCode={info.getValue()} />
            }),
            columnHelper.accessor('name', {
                header: 'Server',
                size: 450,
                cell: (info) => (
                    <Typography variant={'button'} fontFamily={tf2Fonts}>
                        {info.getValue()}
                    </Typography>
                )
            }),
            columnHelper.accessor('map', {
                header: 'Map',
                size: 150,
                cell: (info) => <Typography variant={'body2'}>{cleanMapName(info.getValue())}</Typography>
            }),
            columnHelper.accessor('players', {
                header: 'Players',
                size: 50,
                cell: (info) => (
                    <Typography
                        variant={'body2'}
                    >{`${selectedServers[info.row.index].humans + selectedServers[info.row.index].bots}/${selectedServers[info.row.index].max_players}`}</Typography>
                )
            }),
            columnHelper.accessor('distance', {
                header: 'Dist',
                size: 60,
                meta: {
                    tooltip: 'Approximate distance from you'
                },
                cell: (info) => (
                    <Tooltip title={`Distance in hammer units: ${Math.round((info.getValue() ?? 1) * 52.49)} khu`}>
                        <Typography variant={'caption'}>{`${info.getValue().toFixed(0)}km`}</Typography>
                    </Tooltip>
                )
            }),
            columnHelper.accessor('copy', {
                header: 'Cp',
                size: 30,
                meta: {
                    tooltip: 'Copy to clipboard'
                },
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
            hasPermission(PermissionLevel.Moderator)
                ? columnHelper.display({
                      header: 'Queue',
                      id: 'queue',
                      size: 30,
                      cell: (info) => {
                          const queued = isQueued(info.row.original.server_id);

                          const count = lobbies
                              ? (lobbies.find((value) => {
                                    return value.server_id == info.row.original.server_id;
                                })?.members?.length ?? 0)
                              : 0;

                          return (
                              <Tooltip title="Join/Leave server queue. Number indicates actively queued players. (in testing)">
                                  <IconButton
                                      disabled={false}
                                      color={queued ? 'success' : 'primary'}
                                      onClick={() => {
                                          if (queued) {
                                              leaveQueue([String(info.row.original.server_id)]);
                                          } else {
                                              joinQueue([String(info.row.original.server_id)]);
                                          }
                                      }}
                                  >
                                      <StyledBadge badgeContent={count}>
                                          <GroupsIcon />
                                      </StyledBadge>
                                  </IconButton>
                              </Tooltip>
                          );
                      }
                  })
                : undefined,
            columnHelper.accessor('connect', {
                header: 'Connect',
                size: 100,
                cell: (info) => (
                    <Button
                        fullWidth
                        endIcon={<ChevronRightIcon />}
                        component={Link}
                        href={`steam://run/440//+connect ${selectedServers[info.row.index].ip}:${selectedServers[info.row.index].port}`}
                        variant={'contained'}
                        sx={{ minWidth: 100 }}
                    >
                        Join
                    </Button>
                )
            })
        ].filter((f) => f);
    }, [lobbies, selectedServers, profile]);

    const opts: TableOptions<ServerRow> = {
        data: metaServers,
        columns: columns as ColumnDef<ServerRow>[],
        getCoreRowModel: getCoreRowModel(),
        manualPagination: false,
        autoResetPageIndex: true
    };

    const table = useReactTable(opts);

    if (selectedServers.length === 0) {
        return <Typography textAlign={'center'}>No Servers Matched</Typography>;
    }

    return <DataTable table={table} isLoading={false} />;
};
