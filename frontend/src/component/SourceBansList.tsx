import { useQuery } from "@connectrpc/connect-query";
import HistoryIcon from "@mui/icons-material/History";
import Stack from "@mui/material/Stack";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableContainer from "@mui/material/TableContainer";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import Typography from "@mui/material/Typography";
import type { SourceBanRecord } from "../rpc/ban/v1/ban_pb.ts";
import { querySourceBans } from "../rpc/ban/v1/ban-BanService_connectquery.ts";
import { renderTimestamp } from "../util/time.ts";
import { ContainerWithHeader } from "./ContainerWithHeader";
import { TableCellBool } from "./table/TableCellBool.tsx";

interface SourceBansListProps {
	steamId: bigint;
	isReporter: boolean;
}

export const SourceBansList = ({ steamId, isReporter }: SourceBansListProps) => {
	const { data, isLoading } = useQuery(querySourceBans, { steamId });

	if (!data?.bans || isLoading) {
		return;
	}

	return (
		<ContainerWithHeader title={"External Ban History"} iconLeft={<HistoryIcon />}>
			<Stack spacing={1}>
				<Typography variant={"h5"}>{isReporter ? "Reporter History" : "Suspect History"}</Typography>
				<TableContainer>
					<Table size="small">
						<TableHead>
							<TableRow>
								<TableCell>Created</TableCell>
								<TableCell>Source</TableCell>
								<TableCell>Name</TableCell>
								<TableCell>Reason</TableCell>
								<TableCell>Perm</TableCell>
							</TableRow>
						</TableHead>
						<TableBody>
							{data.bans.map((ban: SourceBanRecord) => {
								return (
									<TableRow key={`ban-${String(ban.createdOn)}`} hover>
										<TableCell>{renderTimestamp(ban.createdOn)}</TableCell>
										<TableCell>{ban.siteName}</TableCell>
										<TableCell>{ban.personaName}</TableCell>
										<TableCell>{ban.reason}</TableCell>
										<TableCellBool enabled={ban.permanent} />
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
