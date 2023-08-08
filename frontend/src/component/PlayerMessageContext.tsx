import React, { useEffect, useState } from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import TableCell from '@mui/material/TableCell';
import TableBody from '@mui/material/TableBody';
import { apiGetMessageContext, PersonMessage } from '../api';
import Box from '@mui/material/Box';
import CircularProgress from '@mui/material/CircularProgress';
import Table from '@mui/material/Table';
import Link from '@mui/material/Link';
import Typography from '@mui/material/Typography';

interface PlayerMessageContextProps {
    playerMessageId: number;
    padding: number;
}

export const PlayerMessageContext = ({
    playerMessageId,
    padding = 3
}: PlayerMessageContextProps) => {
    const [messages, setMessages] = useState<PersonMessage[]>([]);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        apiGetMessageContext(playerMessageId, padding)
            .then((resp) => {
                if (resp.result) {
                    setMessages(resp.result);
                }
            })
            .finally(() => {
                setLoading(false);
            });
    }, [playerMessageId, padding]);

    return (
        <Grid container>
            {loading && (
                <Grid xs={12}>
                    <Box>
                        <CircularProgress color="secondary" />
                    </Box>
                </Grid>
            )}
            {!loading && (
                <>
                    <Grid xs={12}>
                        <TableContainer>
                            <Table size={'small'}>
                                <TableHead>
                                    <TableRow>
                                        <TableCell width={'75px'}>
                                            Server
                                        </TableCell>
                                        <TableCell width={'200px'}>
                                            Name
                                        </TableCell>
                                        <TableCell>Message</TableCell>
                                    </TableRow>
                                </TableHead>
                                <TableBody>
                                    {messages &&
                                        messages.map((m) => {
                                            return (
                                                <TableRow
                                                    key={`chat-msg-${m.person_message_id}`}
                                                    selected={
                                                        playerMessageId ==
                                                        m.person_message_id
                                                    }
                                                >
                                                    <TableCell>
                                                        <Typography
                                                            variant={'body2'}
                                                        >
                                                            {m.server_name}
                                                        </Typography>
                                                    </TableCell>
                                                    <TableCell>
                                                        <Link
                                                            href={`/profile/${m.steam_id}`}
                                                        >
                                                            {m.persona_name}
                                                        </Link>
                                                    </TableCell>
                                                    <TableCell>
                                                        <Typography
                                                            variant={'body1'}
                                                        >
                                                            {m.body}
                                                        </Typography>
                                                    </TableCell>
                                                </TableRow>
                                            );
                                        })}
                                </TableBody>
                            </Table>
                        </TableContainer>
                    </Grid>
                </>
            )}
        </Grid>
    );
};
