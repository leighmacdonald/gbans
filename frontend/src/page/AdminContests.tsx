import React, { useCallback } from 'react';
import { apiContestDelete, apiContests } from '../api';
import Button from '@mui/material/Button';
import NiceModal from '@ebay/nice-modal-react';
import { ModalConfirm, ModalContestEditor } from '../component/modal';
import { LazyTableSimple } from '../component/LazyTableSimple';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import Stack from '@mui/material/Stack';
import ButtonGroup from '@mui/material/ButtonGroup';
import { IconButton } from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import Grid from '@mui/material/Unstable_Grid2';
import { logErr } from '../util/errors';

export const AdminContests = () => {
    const onNewContest = useCallback(async () => {
        await NiceModal.show(ModalContestEditor, {});
    }, []);

    const onDeleteContest = useCallback(async (contest_id: string) => {
        try {
            await apiContestDelete(contest_id);
        } catch (e) {
            logErr(e);
            throw e;
        }
    }, []);

    return (
        <>
            <Grid container marginBottom={3}>
                <Grid xs={12}>
                    <Button
                        variant={'contained'}
                        onClick={onNewContest}
                        color={'success'}
                    >
                        Create New Contest
                    </Button>
                </Grid>
            </Grid>
            <ContainerWithHeader title={'User Submission Contests'}>
                <Stack>
                    <LazyTableSimple
                        fetchData={apiContests}
                        columns={[
                            {
                                sortKey: 'title',
                                label: 'title',
                                tooltip: 'unique contest identifier',
                                align: 'left'
                            },
                            {
                                sortKey: 'description',
                                label: 'description',
                                tooltip: 'description',
                                align: 'left',
                                renderer: (obj) => {
                                    return obj.description.slice(0, 100);
                                }
                            },
                            {
                                sortKey: 'num_entries',
                                label: 'num_entries',
                                tooltip: 'num_entries',
                                align: 'center',
                                sortType: 'number'
                            },
                            {
                                sortKey: 'public',
                                label: 'public',
                                tooltip: 'public',
                                align: 'center',
                                sortType: 'boolean'
                            },
                            {
                                sortKey: 'date_start',
                                sortType: 'date',
                                label: 'date_start',
                                tooltip: 'Starting date',
                                align: 'left',
                                renderer: (obj) => {
                                    return obj.date_start.toISOString();
                                }
                            },
                            {
                                sortKey: 'date_end',
                                sortType: 'date',
                                label: 'date_end',
                                tooltip: 'Ending date',
                                align: 'left'
                            },
                            {
                                virtual: true,
                                virtualKey: 'actions',
                                label: '',
                                tooltip: '',
                                align: 'right',
                                renderer: (obj) => {
                                    return (
                                        <ButtonGroup>
                                            <IconButton color={'warning'}>
                                                <EditIcon />
                                            </IconButton>
                                            <IconButton
                                                color={'error'}
                                                onClick={async () => {
                                                    try {
                                                        await NiceModal.show(
                                                            ModalConfirm,
                                                            {
                                                                title: 'Delete contest?',
                                                                description: `Are you sure you want to delete the contest: ${obj.title}`,
                                                                onConfirm:
                                                                    async () => {
                                                                        await onDeleteContest(
                                                                            obj.contest_id
                                                                        );
                                                                    }
                                                            }
                                                        );
                                                    } catch (e) {
                                                        logErr(e);
                                                    }
                                                }}
                                            >
                                                <DeleteIcon />
                                            </IconButton>
                                        </ButtonGroup>
                                    );
                                }
                            }
                        ]}
                        defaultSortColumn={'date_start'}
                    />
                </Stack>
            </ContainerWithHeader>
        </>
    );
};
