import HistoryIcon from "@mui/icons-material/History";
import Stack from "@mui/material/Stack";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableContainer from "@mui/material/TableContainer";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import Typography from "@mui/material/Typography";
import { useQuery } from "@tanstack/react-query";
import { apiGetSourceBans } from "../api";
import type { sbBanRecord } from "../schema/bans.ts";
import { ContainerWithHeader } from "./ContainerWithHeader";
import { TableCellBool } from "./table/TableCellBool.tsx";

interface SourceBansListProps {
	steam_id: string;
	is_reporter: boolean;
}

export const SourceBansList = ({
	steam_id,
	is_reporter,
}: SourceBansListProps) => {
	const { data: bans } = useQuery({
		queryKey: ["sourcebans", { steam_id }],
		queryFn: async () => {
			return await apiGetSourceBans(steam_id);
		},
	});

	if (!bans) {
		return;
	}

	return (
		<ContainerWithHeader
			title={"External Ban History"}
			iconLeft={<HistoryIcon />}
		>
			<Stack spacing={1}>
				<Typography variant={"h5"}>
					{is_reporter ? "Reporter History" : "Suspect History"}
				</Typography>
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
							{bans.map((ban: sbBanRecord) => {
								return (
									<TableRow key={`ban-${ban.created_on.toDateString()}`} hover>
										<TableCell>{ban.created_on.toDateString()}</TableCell>
										<TableCell>{ban.site_name}</TableCell>
										<TableCell>{ban.persona_name}</TableCell>
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
