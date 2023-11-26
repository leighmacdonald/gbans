import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import MessageIcon from '@mui/icons-material/Message';
import { TablePagination } from '@mui/material';
import Stack from '@mui/material/Stack';
import Grid from '@mui/material/Unstable_Grid2';
import { apiForum, apiGetThreads, Forum, ForumThread } from '../api/forum';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { RowsPerPage } from '../component/LazyTable';
import { logErr } from '../util/errors';

const ForumThreadRow = ({ thread }: { thread: ForumThread }) => {
    return (
        <Grid container>
            <Grid xs>{thread.title}</Grid>
        </Grid>
    );
};

export const ForumPage = () => {
    const { forum_id } = useParams();
    const [forum, setForum] = useState<Forum>();
    const [threads, setThreads] = useState<ForumThread[]>();
    const [count, setCount] = useState(0);
    const [loading, setLoading] = useState(false);
    const [page, setPage] = useState(0);
    const [rowsPerPage, setRowsPerPage] = useState<RowsPerPage>(
        RowsPerPage.TwentyFive
    );
    const id = parseInt(forum_id as string);

    useEffect(() => {
        setLoading(true);
        const abortController = new AbortController();

        apiForum(id, abortController)
            .then((f) => {
                setForum(f);
            })
            .catch((e) => {
                logErr(e);
            })
            .finally(() => {
                setLoading(false);
            });

        return () => abortController.abort();
    }, [id]);

    useEffect(() => {
        apiGetThreads({
            forum_id: id,
            offset: page * rowsPerPage,
            limit: rowsPerPage,
            order_by: 'updated_on',
            desc: true
        }).then((resp) => {
            setThreads(resp.data);
            setCount(resp.count);
        });
    }, [id, page, rowsPerPage]);

    return (
        <ContainerWithHeader
            title={loading ? 'Loading...' : forum?.title ?? 'Forum'}
            iconLeft={<MessageIcon />}
        >
            <Stack>
                {threads?.map((t) => {
                    return (
                        <ForumThreadRow
                            thread={t}
                            key={`ft-${t.forum_thread_id}`}
                        />
                    );
                })}
                <TablePagination
                    rowsPerPage={rowsPerPage}
                    count={count}
                    page={page}
                    onRowsPerPageChange={(event) => {
                        setRowsPerPage(parseInt(event.target.value));
                    }}
                    onPageChange={(_, newPage) => {
                        setPage(newPage);
                    }}
                />
            </Stack>
        </ContainerWithHeader>
    );
};
