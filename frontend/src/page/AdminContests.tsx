import React, { useCallback } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import NiceModal, { useModal } from '@ebay/nice-modal-react';
import DeleteIcon from '@mui/icons-material/Delete';
import EditIcon from '@mui/icons-material/Edit';
import EmojiEventsIcon from '@mui/icons-material/EmojiEvents';
import { IconButton } from '@mui/material';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Grid from '@mui/material/Unstable_Grid2';
import { useTheme } from '@mui/material/styles';
import { apiContestDelete, apiContests, Contest } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { LazyTableSimple } from '../component/LazyTableSimple';
import { ModalConfirm, ModalContestEditor } from '../component/modal';
import { logErr } from '../util/errors';

export const AdminContests = () => {
    const modal = useModal(ModalConfirm);
    const theme = useTheme();

    const onEditContest = useCallback(async (contest_id?: string) => {
        try {
            await NiceModal.show<Contest>(
                ModalContestEditor,
                contest_id ? { contest_id } : {}
            );
        } catch (e) {
            logErr(e);
        }
    }, []);

    const onDeleteContest = useCallback(
        async (contest_id: string) => {
            try {
                await apiContestDelete(contest_id);
                await modal.hide();
            } catch (e) {
                logErr(e);
                throw e;
            }
        },
        [modal]
    );

    return (
        <>
            <Grid container marginBottom={3}>
                <Grid xs={12}>
                    <Button
                        variant={'contained'}
                        onClick={async () => {
                            await onEditContest();
                        }}
                        color={'success'}
                    >
                        Create New Contest
                    </Button>
                </Grid>
            </Grid>
            <ContainerWithHeader
                title={'User Submission Contests'}
                iconLeft={<EmojiEventsIcon />}
            >
                <Stack>
                    <LazyTableSimple
                        fetchData={apiContests}
                        columns={[
                            {
                                sortKey: 'title',
                                label: 'title',
                                tooltip: 'unique contest identifier',
                                align: 'left',
                                renderer: (obj) => {
                                    return (
                                        <Link
                                            component={RouterLink}
                                            variant={'subtitle2'}
                                            to={`/contests/${obj.contest_id}`}
                                            sx={{
                                                color: theme.palette.text
                                                    .primary
                                            }}
                                        >
                                            {obj.title}
                                        </Link>
                                    );
                                }
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
                                label: 'Entries',
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
                                sortKey: 'hide_submissions',
                                label: 'Hide Subs.',
                                tooltip:
                                    'Hide submissions from the public until contest is over',
                                align: 'center',
                                sortType: 'boolean'
                            },
                            {
                                sortKey: 'voting',
                                label: 'Voting',
                                tooltip: 'User entry voting enabled',
                                align: 'center',
                                sortType: 'boolean'
                            },
                            {
                                sortKey: 'down_votes',
                                label: 'Down Votes',
                                tooltip:
                                    'If User entry voting enabled, this will enable/disable the ability to down vote',
                                align: 'center',
                                sortType: 'boolean'
                            },
                            {
                                sortKey: 'date_start',
                                sortType: 'date',
                                label: 'Starting',
                                tooltip: 'Starting date',
                                sortable: true,
                                align: 'left',
                                renderer: (obj) => {
                                    return obj.date_start.toISOString();
                                }
                            },
                            {
                                sortKey: 'date_end',
                                sortType: 'date',
                                sortable: true,
                                label: 'Ending',
                                tooltip: 'Ending date',
                                align: 'left',
                                renderer: (obj) => {
                                    return obj.date_end.toISOString();
                                }
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
                                            <IconButton
                                                color={'warning'}
                                                onClick={async () => {
                                                    try {
                                                        await onEditContest(
                                                            obj.contest_id
                                                        );
                                                    } catch (e) {
                                                        logErr(e);
                                                    }
                                                }}
                                            >
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
                                                        await modal.hide();
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
                        defaultSortColumn={'date_end'}
                        defaultSortDir={'desc'}
                    />
                </Stack>
            </ContainerWithHeader>
        </>
    );
};
