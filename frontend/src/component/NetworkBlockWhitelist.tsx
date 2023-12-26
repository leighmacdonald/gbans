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
import {
    apiDeleteCIDRBlockWhitelist,
    CIDRBlockSource,
    CIDRBlockWhitelist
} from '../api';
import { useCIDRBlocks } from '../hooks/useCIDRBlocks';
import { logErr } from '../util/errors';
import { LoadingPlaceholder } from './LoadingPlaceholder';
import { ModalCIDRBlockEditor, ModalConfirm } from './modal';

export const NetworkBlockWhitelist = () => {
    const { loading, data } = useCIDRBlocks();
    const [newSources, setNewSources] = useState<CIDRBlockWhitelist[]>([]);
    const confirmModal = useModal(ModalConfirm);
    const editorModal = useModal(ModalCIDRBlockEditor);

    const sources = useMemo(() => {
        if (loading) {
            return [];
        }
        return [...newSources, ...(data?.sources ?? [])];
    }, [data?.sources, loading, newSources]);

    const onDelete = useCallback(async (cidr_block_whitelist_id: number) => {
        try {
            const confirmed = await confirmModal.show({
                title: 'Delete IP Whitelist?',
                children: 'This action is permanent'
            });
            if (confirmed) {
                await apiDeleteCIDRBlockWhitelist(cidr_block_whitelist_id);
                await confirmModal.hide();
                await editorModal.hide();
            } else {
                await confirmModal.hide();
            }
        } catch (e) {
            logErr(e);
        }
    }, []);

    const onEdit = useCallback(async (source?: CIDRBlockSource) => {
        try {
            const updated = await NiceModal.show<CIDRBlockWhitelist>(
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
        <Grid container spacing={1}>
            <Grid xs={12}>
                <ButtonGroup>
                    <Button
                        startIcon={<LibraryAddIcon />}
                        variant={'contained'}
                        color={'success'}
                        onClick={async () => {
                            await onEdit();
                        }}
                    >
                        Add New
                    </Button>
                </ButtonGroup>
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
                                            await onDelete(
                                                s.cidr_block_source_id
                                            );
                                        }}
                                    >
                                        Delete
                                    </Button>
                                </ButtonGroup>

                                <Typography variant={'body1'} padding={1}>
                                    {s.name}
                                </Typography>

                                <Typography variant={'body2'} padding={1}>
                                    {s.enabled ? 'Enabled' : 'Disabled'}
                                </Typography>

                                <Typography variant={'body2'} padding={1}>
                                    {s.url}
                                </Typography>
                            </Stack>
                        );
                    })}
                </Grid>
            )}
        </Grid>
    );
};
