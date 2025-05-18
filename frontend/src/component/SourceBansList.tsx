import { useEffect, useState, JSX } from 'react';
import HistoryIcon from '@mui/icons-material/History';
import Stack from '@mui/material/Stack';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Typography from '@mui/material/Typography';
import { apiGetSourceBans } from '../api';
import { sbBanRecord } from '../schema/bans.ts';
import { logErr } from '../util/errors';
import { ContainerWithHeader } from './ContainerWithHeader';

interface SourceBansListProps {
    steam_id: string;
    is_reporter: boolean;
}

export const SourceBansList = ({ steam_id, is_reporter }: SourceBansListProps): JSX.Element => {
    const [bans, setBans] = useState<sbBanRecord[]>([]);

    useEffect(() => {
        apiGetSourceBans(steam_id)
            .then((resp) => {
                setBans(resp);
            })
            .catch(logErr);
    }, [steam_id]);

    if (!bans.length) {
        return <></>;
    }

    return (
        <ContainerWithHeader title={'Suspect SourceBans History'} iconLeft={<HistoryIcon />}>
            <Stack spacing={1}>
                <Typography variant={'h5'}>
                    {is_reporter ? 'Reporter SourceBans History' : 'Suspect SourceBans History'}
                </Typography>
                <TableContainer>
                    <Table size="small">
                        <TableHead>
                            <TableRow>
                                <TableCell>Created</TableCell>
                                <TableCell>Source</TableCell>
                                <TableCell>Name</TableCell>
                                <TableCell>Reason</TableCell>
                                <TableCell>Permanent</TableCell>
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {bans.map((ban) => {
                                return (
                                    <TableRow key={`ban-${ban.ban_id}`} hover>
                                        <TableCell>{ban.created_on.toDateString()}</TableCell>
                                        <TableCell>{ban.site_name}</TableCell>
                                        <TableCell>{ban.persona_name}</TableCell>
                                        <TableCell>{ban.reason}</TableCell>
                                        <TableCell>{ban.permanent ? 'True' : 'False'}</TableCell>
                                    </TableRow>
                                );
                            })}
                        </TableBody>
                    </Table>
                </TableContainer>
            </Stack>
        </ContainerWithHeader>
    );
};
