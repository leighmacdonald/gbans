import Paper from "@mui/material/Paper";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableContainer from "@mui/material/TableContainer";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import { useEffect, useState } from "react";
import { apiGetStats } from "../api";
import type { DatabaseStats } from "../schema/stats.ts";
import { logErr } from "../util/errors";

export const StatsPanel = () => {
	const [stats, setStats] = useState<DatabaseStats>({
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
		servers_total: 0,
	});

	useEffect(() => {
		apiGetStats()
			.then((response) => {
				setStats(response);
			})
			.catch(logErr);
	}, []);

	return (
		<TableContainer component={Paper}>
			<Table aria-label="customized table">
				<TableHead>
					<TableRow>
						<TableCell>Metric</TableCell>
						<TableCell align="right">Value</TableCell>
					</TableRow>
				</TableHead>
				<TableBody>
					<TableRow>
						<TableCell component="th" scope="row">
							Bans Total
						</TableCell>
						<TableCell align="right">{stats.bans}</TableCell>
					</TableRow>
					<TableRow>
						<TableCell component="th" scope="row">
							Bans 1 Week
						</TableCell>
						<TableCell align="right">{stats.bans_week}</TableCell>
					</TableRow>
					<TableRow>
						<TableCell component="th" scope="row">
							Bans 1 Month
						</TableCell>
						<TableCell align="right">{stats.bans_month}</TableCell>
					</TableRow>
					<TableRow>
						<TableCell component="th" scope="row">
							Bans 3 Months
						</TableCell>
						<TableCell align="right">{stats.bans_3month}</TableCell>
					</TableRow>
					<TableRow>
						<TableCell component="th" scope="row">
							Bans 6 Months
						</TableCell>
						<TableCell align="right">{stats.bans_6month}</TableCell>
					</TableRow>
					<TableRow>
						<TableCell component="th" scope="row">
							Bans 1 Year
						</TableCell>
						<TableCell align="right">{stats.bans_year}</TableCell>
					</TableRow>
					<TableRow>
						<TableCell component="th" scope="row">
							CIDR Bans
						</TableCell>
						<TableCell align="right">{stats.bans_cidr}</TableCell>
					</TableRow>
					<TableRow>
						<TableCell component="th" scope="row">
							Servers (Alive)
						</TableCell>
						<TableCell align="right">
							{stats.servers_total} ({stats.servers_alive})
						</TableCell>
					</TableRow>
				</TableBody>
			</Table>
		</TableContainer>
	);
};
