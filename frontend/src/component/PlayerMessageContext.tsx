import Box from '@mui/material/Box';
import CircularProgress from '@mui/material/CircularProgress';
import Grid from '@mui/material/Grid';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Typography from '@mui/material/Typography';
import { useQuery } from '@tanstack/react-query';
import { apiGetMessageContext } from '../api';
import { TextLink } from './TextLink.tsx';

interface PlayerMessageContextProps {
    playerMessageId: number;
    padding: number;
}

export const PlayerMessageContext = ({ playerMessageId, padding = 3 }: PlayerMessageContextProps) => {
    const { data: messages, isLoading } = useQuery({
        queryKey: ['messageContext', playerMessageId],
        queryFn: async () => {
            return await apiGetMessageContext(playerMessageId, padding);
        }
    });

    return (
        <Grid container>
            {isLoading && (
                <Grid size={{ xs: 12 }}>
                    <Box>
                        <CircularProgress color="secondary" />
                    </Box>
                </Grid>
            )}
            {!isLoading && (
                <>
                    <Grid size={{ xs: 12 }}>
                        <TableContainer>
                            <Table size={'small'}>
                                <TableHead>
                                    <TableRow>
                                        <TableCell width={'75px'}>Server</TableCell>
                                        <TableCell width={'200px'}>Name</TableCell>
                                        <TableCell>Message</TableCell>
                                    </TableRow>
                                </TableHead>
                                <TableBody>
                                    {messages &&
                                        messages.map((m) => {
                                            return (
                                                <TableRow
                                                    key={`chat-msg-${m.person_message_id}`}
                                                    selected={playerMessageId == m.person_message_id}
                                                >
                                                    <TableCell>
                                                        <Typography variant={'body2'}>{m.server_name}</Typography>
                                                    </TableCell>
                                                    <TableCell>
                                                        <TextLink
                                                            to={`/profile/$steamId`}
                                                            params={{ steamId: m.steam_id }}
                                                        >
                                                            {m.persona_name}
                                                        </TextLink>
                                                    </TableCell>
                                                    <TableCell>
                                                        <Typography variant={'body1'}>{m.body}</Typography>
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
