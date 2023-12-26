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
import { apiDeleteCIDRBlockSource, CIDRBlockSource } from '../api';
import { useCIDRBlocks } from '../hooks/useCIDRBlocks';
import { logErr } from '../util/errors';
import { LoadingPlaceholder } from './LoadingPlaceholder';
import { VCenterBox } from './VCenterBox';
import { ModalCIDRBlockEditor, ModalConfirm } from './modal';
import { CIDRWhitelistSection } from './table/CIDRWhitelistSection';

export const NetworkBlockSources = () => {
    const { loading, data } = useCIDRBlocks();
    const [newSources, setNewSources] = useState<CIDRBlockSource[]>([]);
    const confirmModal = useModal(ModalConfirm);
    const editorModal = useModal(ModalCIDRBlockEditor);

    const sources = useMemo(() => {
        if (loading) {
            return [];
        }
        return [...newSources, ...(data?.sources ?? [])];
    }, [data?.sources, loading, newSources]);

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
            const updated = await NiceModal.show<CIDRBlockSource>(
                ModalCIDRBlockEditor,
                {
                    source
                }
            );

            setNewSources((prevState) => {
                return [
                    updated,
                    ...prevState.filter(
                        (s) =>
                            s.cidr_block_source_id !=
                            updated.cidr_block_source_id
                    )
                ];
            });
        } catch (e) {
            logErr(e);
        }
    }, []);

    return (
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
                {loading ? (
                    <LoadingPlaceholder />
                ) : (
                    <Grid xs={12}>
                        {sources.map((s) => {
                            return (
                                <Stack
                                    spacing={1}
                                    direction={'row'}
                                    key={`cidr-source-${s.cidr_block_source_id}`}
                                >
                                    <ButtonGroup size={'small'}>
                                        <Button
                                            startIcon={<EditIcon />}
                                            variant={'contained'}
                                            color={'warning'}
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
                                                await onDeleteSource(
                                                    s.cidr_block_source_id
                                                );
                                            }}
                                        >
                                            Delete
                                        </Button>
                                    </ButtonGroup>

                                    <VCenterBox>
                                        <Typography variant={'body1'}>
                                            {s.name}
                                        </Typography>
                                    </VCenterBox>
                                    <VCenterBox>
                                        <Typography variant={'body2'}>
                                            {s.enabled ? 'Enabled' : 'Disabled'}
                                        </Typography>
                                    </VCenterBox>
                                    <VCenterBox>
                                        <Typography variant={'body2'}>
                                            {s.url}
                                        </Typography>
                                    </VCenterBox>
                                </Stack>
                            );
                        })}
                    </Grid>
                )}
            </Grid>
            <CIDRWhitelistSection rows={data?.whitelist ?? []} />
        </Stack>
    );
};
