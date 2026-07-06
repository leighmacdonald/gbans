import { useQuery } from "@connectrpc/connect-query";
import Box from "@mui/material/Box";
import CircularProgress from "@mui/material/CircularProgress";
import Grid from "@mui/material/Grid";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableContainer from "@mui/material/TableContainer";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import Typography from "@mui/material/Typography";
import { useMemo } from "react";
import { queryContext } from "../rpc/chat/v1/chat-ChatService_connectquery.ts";
import { servers } from "../rpc/servers/v1/servers-ServersService_connectquery.ts";
import { emptyOrNullString } from "../util/types.ts";
import { TextLink } from "./TextLink.tsx";

interface PlayerMessageContextProps {
	playerMessageId: string;
	padding: number;
}

export const PlayerMessageContext = ({ playerMessageId, padding = 3 }: PlayerMessageContextProps) => {
	const { data: serversResp, isLoading: isLoadingServers } = useQuery(servers);
	const { data, isLoading } = useQuery(queryContext, { personMessageId: playerMessageId, padding });

	const serverName = useMemo(() => {
		if (!data?.messages) {
			return "";
		}
		return serversResp?.servers.find((s) => s.serverId === data?.messages[0].serverId)?.serverName ?? "";
	}, [serversResp, data?.messages]);

	return (
		<Grid container>
			{isLoading && (
				<Grid size={{ xs: 12 }}>
					<Box>
						<CircularProgress color="secondary" />
					</Box>
				</Grid>
			)}
			{!isLoading && !isLoadingServers && (
				<Grid size={{ xs: 12 }}>
					<TableContainer>
						<Table size={"small"}>
							<TableHead>
								<TableRow>
									<TableCell width={"75px"}>Server</TableCell>
									<TableCell width={"200px"}>Name</TableCell>
									<TableCell>Message</TableCell>
								</TableRow>
							</TableHead>
							<TableBody>
								{data?.messages?.map((m) => {
									return (
										<TableRow
											key={`chat-msg-${m.personMessageId}`}
											selected={playerMessageId === m.personMessageId}
										>
											<TableCell>
												<Typography variant={"body2"}>{serverName}</Typography>
											</TableCell>
											<TableCell>
												<TextLink
													to={`/profile/$steamId`}
													params={{ steamId: String(m.steamId) }}
												>
													{emptyOrNullString(m.personaName) ? m.steamId : m.personaName}
												</TextLink>
											</TableCell>
											<TableCell>
												<Typography variant={"body1"}>{m.body}</Typography>
											</TableCell>
										</TableRow>
									);
								})}
							</TableBody>
						</Table>
					</TableContainer>
				</Grid>
			)}
		</Grid>
	);
};
