import React, { useCallback, useMemo, useState } from 'react';
import NiceModal, { useModal } from '@ebay/nice-modal-react';
import DeleteIcon from '@mui/icons-material/Delete';
import EditIcon from '@mui/icons-material/Edit';
import LibraryAddIcon from '@mui/icons-material/LibraryAdd';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { apiDeleteCIDRBlockWhitelist, CIDRBlockWhitelist } from '../../api';
import { logErr } from '../../util/errors';
import { renderDateTime } from '../../util/text';
import { VCenterBox } from '../VCenterBox';
import { ModalCIDRWhitelistEditor, ModalConfirm } from '../modal';
import { LazyTable, Order, RowsPerPage } from './LazyTable';
import { compare, stableSort } from './LazyTableSimple';

export const CIDRWhitelistSection = ({
    rows
}: {
    rows: CIDRBlockWhitelist[];
}) => {
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] =
        useState<keyof CIDRBlockWhitelist>('address');
    const [page, setPage] = useState(0);
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        RowsPerPage.TwentyFive
    );
    const [deletedIds, setDeletedIds] = useState<number[]>([]);
    const [newWhitelist, setNewWhitelist] = useState<CIDRBlockWhitelist[]>([]);
    const confirmModal = useModal(ModalConfirm);
    const editorModal = useModal(ModalCIDRWhitelistEditor);

    const whitelists = useMemo(() => {
        return [...newWhitelist, ...(rows ?? [])].filter(
            (w) => !deletedIds.includes(w.cidr_block_whitelist_id)
        );
    }, [newWhitelist, rows, deletedIds]);

    const sorted = useMemo(() => {
        return stableSort(whitelists, compare(sortOrder, sortColumn)).slice(
            page * rowPerPageCount,
            page * rowPerPageCount + rowPerPageCount
        );
    }, [whitelists, sortOrder, sortColumn, page, rowPerPageCount]);

    const onEdit = useCallback(async (source?: CIDRBlockWhitelist) => {
        try {
            const updated = await NiceModal.show<CIDRBlockWhitelist>(
                ModalCIDRWhitelistEditor,
                {
                    source
                }
            );

            setNewWhitelist((prevState) => {
                return [
                    updated,
                    ...prevState.filter(
                        (s) =>
                            s.cidr_block_whitelist_id !=
                            updated.cidr_block_whitelist_id
                    )
                ];
            });
        } catch (e) {
            logErr(e);
        }
    }, []);

    const onDeleteWhitelist = useCallback(
        async (cidr_block_whitelist_id: number) => {
            try {
                const confirmed = await confirmModal.show({
                    title: 'Delete CIDR Whitelist?',
                    children: 'This action is permanent'
                });
                if (confirmed) {
                    await apiDeleteCIDRBlockWhitelist(cidr_block_whitelist_id);
                    setDeletedIds((prevState) => {
                        return [...prevState, cidr_block_whitelist_id];
                    });
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

    return (
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
                        <Typography variant={'h6'}>
                            CIDR/IP Whitelists
                        </Typography>
                    </VCenterBox>
                </Stack>
            </Grid>
            <Grid xs={12}>
                <LazyTable<CIDRBlockWhitelist>
                    columns={[
                        {
                            label: 'ID',
                            align: 'left',
                            sortable: true,
                            sortKey: 'cidr_block_whitelist_id',
                            tooltip: 'Whitelisted IP Address',
                            renderer: (obj) => {
                                return (
                                    <Typography variant={'body2'}>
                                        {obj.cidr_block_whitelist_id}
                                    </Typography>
                                );
                            }
                        },
                        {
                            label: 'Address',
                            align: 'left',
                            sortable: true,
                            sortKey: 'address',
                            tooltip: 'Whitelisted IP Address',
                            renderer: (obj) => {
                                return (
                                    <Typography variant={'body1'}>
                                        {obj.address}
                                    </Typography>
                                );
                            }
                        },
                        {
                            label: 'Created',
                            sortable: true,
                            sortKey: 'created_on',
                            sortType: 'date',
                            tooltip: 'Whitelisted address',
                            renderer: (obj) => {
                                return (
                                    <Typography variant={'body1'}>
                                        {renderDateTime(obj.created_on)}
                                    </Typography>
                                );
                            }
                        },
                        {
                            label: 'Updated',
                            sortable: true,
                            sortKey: 'updated_on',
                            sortType: 'date',
                            tooltip: 'Whitelisted address',
                            renderer: (obj) => {
                                return (
                                    <Typography variant={'body1'}>
                                        {renderDateTime(obj.updated_on)}
                                    </Typography>
                                );
                            }
                        },
                        {
                            label: 'Actions',
                            sortable: false,
                            virtual: true,
                            tooltip: 'Mod actions',
                            renderer: (obj) => {
                                return (
                                    <ButtonGroup>
                                        <Button
                                            startIcon={<EditIcon />}
                                            variant={'contained'}
                                            color={'warning'}
                                            onClick={async () => {
                                                await onEdit(obj);
                                            }}
                                        >
                                            Edit
                                        </Button>
                                        <Button
                                            startIcon={<DeleteIcon />}
                                            variant={'contained'}
                                            color={'error'}
                                            onClick={async () => {
                                                await onDeleteWhitelist(
                                                    obj.cidr_block_whitelist_id
                                                );
                                            }}
                                        >
                                            Delete
                                        </Button>
                                    </ButtonGroup>
                                );
                            }
                        }
                    ]}
                    loading={false}
                    rows={sorted}
                    rowsPerPage={rowPerPageCount}
                    page={page}
                    showPager={true}
                    count={sorted.length}
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
                        event: React.ChangeEvent<
                            HTMLInputElement | HTMLTextAreaElement
                        >
                    ) => {
                        setRowPerPageCount(parseInt(event.target.value, 10));
                        setPage(0);
                    }}
                />
            </Grid>
        </Grid>
    );
};
