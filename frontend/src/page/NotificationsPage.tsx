import React, { useEffect, useState } from 'react';
import DeleteSweepIcon from '@mui/icons-material/DeleteSweep';
import DoneAllIcon from '@mui/icons-material/DoneAll';
import EmailIcon from '@mui/icons-material/Email';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import Checkbox from '@mui/material/Checkbox';
import Pagination from '@mui/material/Pagination';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid2 from '@mui/material/Unstable_Grid2';
import { parseISO } from 'date-fns';
import { UserNotification } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { RowsPerPage } from '../component/LazyTable';
import { useNotifications } from '../contexts/NotificationsCtx';
import { renderDateTime } from '../util/text';

interface CBProps {
    id: number;
}

const CB = ({ id }: CBProps) => {
    const [checked, setChecked] = useState(false);
    const { setSelectedIds } = useNotifications();
    return (
        <Checkbox
            checked={checked}
            onChange={(_, checked) => {
                setChecked(checked);
                setSelectedIds((prevState) => {
                    if (checked && !prevState.includes(id)) {
                        prevState.push(id);
                    } else {
                        const index = prevState.indexOf(id, 0);
                        if (index > -1) {
                            prevState.splice(index, 1);
                        }
                    }
                    return prevState;
                });
            }}
        />
    );
};

export const NotificationsPage = () => {
    const { notifications } = useNotifications();
    const [page, setPage] = useState<number>(0);
    const [visible, setVisible] = useState<UserNotification[]>([]);

    useEffect(() => {
        setVisible(
            notifications.slice(
                page * RowsPerPage.Fifty,
                page * RowsPerPage.Fifty + RowsPerPage.Fifty
            )
        );
    }, [notifications, page]);
    return (
        <Grid2 container spacing={2}>
            <Grid2 xs={3}>
                <ContainerWithHeader title={'Manage'}>
                    <ButtonGroup orientation={'vertical'} variant="contained">
                        <Button startIcon={<DoneAllIcon />} color={'primary'}>
                            Mark All Read
                        </Button>
                        <Button startIcon={<DeleteSweepIcon />} color={'error'}>
                            Delete Selected
                        </Button>
                    </ButtonGroup>
                </ContainerWithHeader>
            </Grid2>
            <Grid2 xs={9}>
                <ContainerWithHeader
                    iconLeft={<EmailIcon />}
                    title={`Notifications (${notifications.length})`}
                    marginTop={0}
                >
                    <Box>
                        {visible.map((n) => {
                            return (
                                <Paper
                                    elevation={1}
                                    key={n.person_notification_id}
                                >
                                    <Stack
                                        direction={'row'}
                                        justifyContent="left"
                                        alignItems="center"
                                        spacing={1}
                                    >
                                        <CB id={n.person_notification_id} />

                                        <Typography variant={'button'}>
                                            {renderDateTime(
                                                parseISO(n.created_on)
                                            )}
                                        </Typography>

                                        <Typography
                                            variant={'body1'}
                                            textOverflow={'ellipsis'}
                                        >
                                            {n.message.substring(0, 200)}
                                        </Typography>
                                    </Stack>
                                </Paper>
                            );
                        })}
                    </Box>
                    <Box paddingTop={0} marginTop={0} paddingBottom={2}>
                        <Pagination
                            onChange={(_, page) => {
                                setPage(page - 1);
                            }}
                            count={
                                notifications
                                    ? Math.ceil(
                                          Math.max(
                                              notifications.length /
                                                  RowsPerPage.Fifty,
                                              1
                                          )
                                      )
                                    : 1
                            }
                            color="primary"
                        />
                    </Box>
                </ContainerWithHeader>
            </Grid2>
        </Grid2>
    );
};
