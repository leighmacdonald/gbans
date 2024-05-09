import { useCallback, useMemo, useState } from 'react';
import NiceModal, { useModal } from '@ebay/nice-modal-react';
import DeleteIcon from '@mui/icons-material/Delete';
import EditIcon from '@mui/icons-material/Edit';
import LibraryAddIcon from '@mui/icons-material/LibraryAdd';
import WifiOffIcon from '@mui/icons-material/WifiOff';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import Stack from '@mui/material/Stack';
import TableCell from '@mui/material/TableCell';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, useRouteContext } from '@tanstack/react-router';
import { createColumnHelper, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { z } from 'zod';
import {
    apiDeleteCIDRBlockSource,
    apiGetCIDRBlockLists,
    CIDRBlockSource,
    CIDRBlockWhitelist,
    PermissionLevel
} from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { DataTable } from '../component/DataTable.tsx';
import { LoadingPlaceholder } from '../component/LoadingPlaceholder.tsx';
import { TableHeadingCell } from '../component/TableHeadingCell.tsx';
import { VCenterBox } from '../component/VCenterBox.tsx';
import { ModalCIDRBlockEditor, ModalConfirm } from '../component/modal';
import { logErr } from '../util/errors.ts';
import { commonTableSearchSchema } from '../util/table.ts';
import { renderDate } from '../util/text.tsx';

const ipHistorySearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z
        .enum(['person_connection_id', 'steam_id', 'created_on', 'ip_addr', 'server_id'])
        .catch('person_connection_id')
});

export const Route = createFileRoute('/_mod/admin/network/cidrblocks')({
    component: AdminNetworkCIDRBlocks,
    validateSearch: (search) => ipHistorySearchSchema.parse(search)
});

function AdminNetworkCIDRBlocks() {
    const { page, rows, sortOrder, sortColumn } = Route.useSearch();
    const { data: blockLists, isLoading } = useQuery({
        queryKey: ['cidrBlockLists', { page, rows, sortOrder, sortColumn }],
        queryFn: async () => {
            return await apiGetCIDRBlockLists();
        }
    });

    const [newSources, setNewSources] = useState<CIDRBlockSource[]>([]);
    const confirmModal = useModal(ModalConfirm);
    const editorModal = useModal(ModalCIDRBlockEditor);
    const { hasPermission } = useRouteContext({ from: '/_mod/admin/network/cidrblocks' });
    //const confirmModalWhitelist = useModal(ModalConfirm);
    //const editorModalWhitelist = useModal(ModalCIDRWhitelistEditor);

    // const onEdit = useCallback(async (source?: CIDRBlockWhitelist) => {
    //     try {
    //         const updated = await NiceModal.show<CIDRBlockWhitelist>(ModalCIDRWhitelistEditor, {
    //             source
    //         });
    //
    //         setNewWhitelist((prevState) => {
    //             return [updated, ...prevState.filter((s) => s.cidr_block_whitelist_id != updated.cidr_block_whitelist_id)];
    //         });
    //     } catch (e) {
    //         logErr(e);
    //     }
    // }, []);
    //
    // const onDeleteWhitelist = useCallback(
    //     async (cidr_block_whitelist_id: number) => {
    //         try {
    //             const confirmed = await confirmModalWhitelist.show({
    //                 title: 'Delete CIDR Whitelist?',
    //                 children: 'This action is permanent'
    //             });
    //             if (confirmed) {
    //                 await apiDeleteCIDRBlockWhitelist(cidr_block_whitelist_id);
    //                 setDeletedIds((prevState) => {
    //                     return [...prevState, cidr_block_whitelist_id];
    //                 });
    //                 await confirmModalWhitelist.hide();
    //                 await editorModal.hide();
    //             } else {
    //                 await confirmModalWhitelist.hide();
    //             }
    //         } catch (e) {
    //             logErr(e);
    //         }
    //     },
    //     [confirmModalWhitelist, editorModal]
    // );

    const sources = useMemo(() => {
        if (isLoading) {
            return [];
        }
        return [...newSources, ...(blockLists?.sources ?? [])];
    }, [blockLists?.sources, isLoading, newSources]);

    const onDeleteSource = useCallback(
        async (cidr_block_source_id: number) => {
            try {
                const confirmed = await confirmModal.show({
                    title: 'Delete CIDR Block Source?',
                    children: 'This action is permanent'
                });
                if (confirmed) {
                    await apiDeleteCIDRBlockSource(cidr_block_source_id);
                    await confirmModal.hide();
                    await editorModal.hide();
                } else {
                    await confirmModal.hide();
                }
            } catch (e) {
                logErr(e);
            }
        },
        [confirmModal, editorModal]
    );

    const onEdit = useCallback(async (source?: CIDRBlockSource) => {
        try {
            const updated = await NiceModal.show<CIDRBlockSource>(ModalCIDRBlockEditor, {
                source
            });

            setNewSources((prevState) => {
                return [updated, ...prevState.filter((s) => s.cidr_block_source_id != updated.cidr_block_source_id)];
            });
        } catch (e) {
            logErr(e);
        }
    }, []);

    return (
        <ContainerWithHeader title="Admin Network CIDR" iconLeft={<WifiOffIcon />}>
            <Stack spacing={2}>
                <Grid container spacing={1}>
                    <Grid xs={12}>
                        <Stack direction={'row'} spacing={1}>
                            <ButtonGroup size={'small'}>
                                <Button
                                    startIcon={<LibraryAddIcon />}
                                    variant={'contained'}
                                    color={'success'}
                                    onClick={async () => {
                                        await onEdit();
                                    }}
                                >
                                    Add CIDR Source
                                </Button>
                            </ButtonGroup>
                            <VCenterBox>
                                <Typography variant={'h6'} textAlign={'right'}>
                                    CIDR Blocklists
                                </Typography>
                            </VCenterBox>
                        </Stack>
                    </Grid>
                    {isLoading ? (
                        <LoadingPlaceholder />
                    ) : (
                        <Grid xs={12}>
                            {sources.map((s) => {
                                return (
                                    <Stack spacing={1} direction={'row'} key={`cidr-source-${s.cidr_block_source_id}`}>
                                        <ButtonGroup size={'small'} disabled={!hasPermission(PermissionLevel.Admin)}>
                                            <Button
                                                startIcon={<EditIcon />}
                                                variant={'contained'}
                                                color={'warning'}
                                                disabled={!hasPermission(PermissionLevel.Admin)}
                                                onClick={async () => {
                                                    await onEdit(s);
                                                }}
                                            >
                                                Edit
                                            </Button>
                                            <Button
                                                startIcon={<DeleteIcon />}
                                                variant={'contained'}
                                                color={'error'}
                                                onClick={async () => {
                                                    await onDeleteSource(s.cidr_block_source_id);
                                                }}
                                            >
                                                Delete
                                            </Button>
                                        </ButtonGroup>

                                        <VCenterBox>
                                            <Typography variant={'body1'}>{s.name}</Typography>
                                        </VCenterBox>
                                        <VCenterBox>
                                            <Typography variant={'body2'}>
                                                {s.enabled ? 'Enabled' : 'Disabled'}
                                            </Typography>
                                        </VCenterBox>
                                        <VCenterBox>
                                            <Typography variant={'body2'}>{s.url}</Typography>
                                        </VCenterBox>
                                    </Stack>
                                );
                            })}
                        </Grid>
                    )}
                </Grid>

                <Grid container spacing={1}>
                    <Grid xs={12}>
                        <Stack direction={'row'} spacing={1}>
                            <ButtonGroup size={'small'}>
                                <Button
                                    startIcon={<LibraryAddIcon />}
                                    variant={'contained'}
                                    color={'success'}
                                    onClick={async () => {
                                        await onEdit();
                                    }}
                                >
                                    Add Whitelist
                                </Button>
                            </ButtonGroup>
                            <VCenterBox>
                                <Typography variant={'h6'}>CIDR/IP Whitelists</Typography>
                            </VCenterBox>
                        </Stack>
                    </Grid>
                    <Grid xs={12}>
                        <WhitelistTable whitelist={blockLists?.whitelist ?? []} isLoading={isLoading} />
                    </Grid>
                </Grid>
            </Stack>
        </ContainerWithHeader>
    );
}

const columnHelper = createColumnHelper<CIDRBlockWhitelist>();

const WhitelistTable = ({ whitelist, isLoading }: { whitelist: CIDRBlockWhitelist[]; isLoading: boolean }) => {
    const columns = [
        columnHelper.accessor('cidr_block_whitelist_id', {
            header: () => <TableHeadingCell name={'ID'} />,
            cell: (info) => <Typography>{info.getValue()}</Typography>
        }),
        columnHelper.accessor('address', {
            header: () => <TableHeadingCell name={'Address'} />,
            cell: (info) => (
                <TableCell>
                    <Typography>{info.getValue()}</Typography>
                </TableCell>
            )
        }),
        columnHelper.accessor('created_on', {
            header: () => <TableHeadingCell name={'IP Address'} />,
            cell: (info) => (
                <TableCell>
                    <Typography>{renderDate(info.getValue())}</Typography>
                </TableCell>
            )
        }),
        columnHelper.accessor('updated_on', {
            header: () => <TableHeadingCell name={'Server'} />,
            cell: (info) => (
                <TableCell>
                    <Typography>{renderDate(info.getValue())}</Typography>
                </TableCell>
            )
        })
    ];

    const table = useReactTable({
        data: whitelist,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        manualPagination: true,
        autoResetPageIndex: true
    });

    return <DataTable table={table} isLoading={isLoading} />;
};
