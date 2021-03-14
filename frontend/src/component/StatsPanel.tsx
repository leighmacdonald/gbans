import React, {useEffect} from 'react';
import {apiGetStats, DatabaseStats} from '../util/api';
import {Paper, Table, TableBody, TableContainer, TableHead, TableRow} from '@material-ui/core';
import {StyledTableCell, StyledTableRow} from './Tables';

export const StatsPanel = () => {
    const [stats, setStats] = React.useState<DatabaseStats>({
        bans: 0,
        appeals_closed: 0,
        appeals_open: 0,
        bans_3month: 0,
        bans_6month: 0,
        bans_cidr: 0,
        bans_day: 0,
        bans_month: 0,
        bans_week: 0,
        bans_year: 0,
        filtered_words: 0,
        servers_alive: 0,
        servers_total: 0
    });

    useEffect(() => {
        const loadStats = async () => {
            try {
                const resp = await apiGetStats();
                setStats(resp);
            } catch (e) {
                console.log(`"Failed to get stats: ${e}`);
            }
        };
        loadStats();
    }, []);
    return (
        <TableContainer component={Paper}>
            <Table aria-label="customized table">
                <TableHead>
                    <TableRow>
                        <StyledTableCell>Metric</StyledTableCell>
                        <StyledTableCell align="right">Value</StyledTableCell>
                    </TableRow>
                </TableHead>
                <TableBody>
                    <StyledTableRow>
                        <StyledTableCell component="th" scope="row">
                            Bans Total
                        </StyledTableCell>
                        <StyledTableCell align="right">{stats.bans}</StyledTableCell>
                    </StyledTableRow>
                    <StyledTableRow>
                        <StyledTableCell component="th" scope="row">
                            Bans 1 Week
                        </StyledTableCell>
                        <StyledTableCell align="right">{stats.bans_week}</StyledTableCell>
                    </StyledTableRow>
                    <StyledTableRow>
                        <StyledTableCell component="th" scope="row">
                            Bans 1 Month
                        </StyledTableCell>
                        <StyledTableCell align="right">{stats.bans_month}</StyledTableCell>
                    </StyledTableRow>
                    <StyledTableRow>
                        <StyledTableCell component="th" scope="row">
                            Bans 3 Months
                        </StyledTableCell>
                        <StyledTableCell align="right">{stats.bans_3month}</StyledTableCell>
                    </StyledTableRow>
                    <StyledTableRow>
                        <StyledTableCell component="th" scope="row">
                            Bans 6 Months
                        </StyledTableCell>
                        <StyledTableCell align="right">{stats.bans_6month}</StyledTableCell>
                    </StyledTableRow>
                    <StyledTableRow>
                        <StyledTableCell component="th" scope="row">
                            Bans 1 Year
                        </StyledTableCell>
                        <StyledTableCell align="right">{stats.bans_year}</StyledTableCell>
                    </StyledTableRow>
                    <StyledTableRow>
                        <StyledTableCell component="th" scope="row">
                            CIDR Bans
                        </StyledTableCell>
                        <StyledTableCell align="right">{stats.bans_cidr}</StyledTableCell>
                    </StyledTableRow>
                    <StyledTableRow>
                        <StyledTableCell component="th" scope="row">
                            Appeals (Open)
                        </StyledTableCell>
                        <StyledTableCell align="right">{stats.appeals_open}</StyledTableCell>
                    </StyledTableRow>
                    <StyledTableRow>
                        <StyledTableCell component="th" scope="row">
                            Appeals (Closed)
                        </StyledTableCell>
                        <StyledTableCell align="right">{stats.appeals_closed}</StyledTableCell>
                    </StyledTableRow>
                    <StyledTableRow>
                        <StyledTableCell component="th" scope="row">
                            Servers (Alive)
                        </StyledTableCell>
                        <StyledTableCell align="right">
                            {stats.servers_total} ({stats.servers_alive})
                        </StyledTableCell>
                    </StyledTableRow>
                </TableBody>
            </Table>
        </TableContainer>
    );
};
