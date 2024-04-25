import { useState } from 'react';
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
import { createLazyFileRoute } from '@tanstack/react-router';
import { parseISO } from 'date-fns';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { useNotifications } from '../hooks/useNotifications';
import { useNotificationsCtx } from '../hooks/useNotificationsCtx.ts';
import { RowsPerPage } from '../util/table.ts';
import { renderDateTime } from '../util/text.tsx';

export const Route = createLazyFileRoute('/notifications')({
    component: NotificationsPage
});

interface CBProps {
    id: number;
}

const CB = ({ id }: CBProps) => {
    const [checked, setChecked] = useState(false);
    const { setSelectedIds } = useNotificationsCtx();
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

function NotificationsPage() {
    const [page, setPage] = useState<number>(0);

    const { data, count } = useNotifications({
        limit: RowsPerPage.TwentyFive,
        desc: true,
        deleted: false,
        offset: page * RowsPerPage.TwentyFive
    });

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
                    title={`Notifications (${count})`}
                    marginTop={0}
                >
                    <Box>
                        {data.map((n) => {
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
                            count={
                                count > 0
                                    ? Math.ceil(count / RowsPerPage.TwentyFive)
                                    : 0
                            }
                            page={page}
                            onChange={(_, newPage) => {
                                setPage(newPage);
                            }}
                        />
                    </Box>
                </ContainerWithHeader>
            </Grid2>
        </Grid2>
    );
}
