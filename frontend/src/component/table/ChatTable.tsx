import { type Timestamp, timestampDate } from "@bufbuild/protobuf/wkt";
import { useQuery } from "@connectrpc/connect-query";
import Button from "@mui/material/Button";
import Typography from "@mui/material/Typography";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useMemo } from "react";
import type { Message } from "../../rpc/chat/v1/chat_pb.ts";
import { query } from "../../rpc/chat/v1/chat-ChatService_connectquery.ts";
import { stringToColour } from "../../util/colours.ts";
import { PersonCell } from "../PersonCell.tsx";
import { createDefaultTableOptions } from "./options.ts";
import { SortableTable } from "./SortableTable.tsx";
import { TableCellRelativeDateField } from "./TableCellRelativeDateField.tsx";

const columnHelper = createMRTColumnHelper<Message>();
const defaultOptions = createDefaultTableOptions<Message>();

export const ChatTable = ({ steamId }: { steamId: bigint }) => {
	const { data, isLoading, isError } = useQuery(query, {
		steamId: steamId,
		filter: { limit: 2500n, orderBy: "person_message_id", desc: true },
	});

	const columns = useMemo(
		() => [
			columnHelper.accessor("serverId", {
				header: "Server",
				grow: false,
				Cell: ({ row }) => (
					<Button
						variant="text"
						sx={{
							color: stringToColour(row.original.serverName),
						}}
					>
						{row.original.serverName}
					</Button>
				),
			}),

			columnHelper.accessor("createdOn", {
				header: "Created",
				grow: false,
				Cell: ({ cell }) => <TableCellRelativeDateField date={timestampDate(cell.getValue() as Timestamp)} />,
			}),

			columnHelper.accessor("personaName", {
				header: "Name",
				grow: false,
				Cell: ({ row }) => (
					<PersonCell
						steamId={row.original.steamId}
						avatarHash={row.original.avatarHash}
						personaName={row.original.personaName}
					/>
				),
			}),

			columnHelper.accessor("body", {
				header: "Message",
				grow: true,
				Cell: ({ cell }) => (
					<Typography padding={0} variant={"body1"}>
						{cell.getValue() as string}
					</Typography>
				),
			}),
		],
		[],
	);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data?.messages ?? [],
		rowCount: Number(data?.count ?? 0),
		enableFilters: true,
		enableRowActions: false,
		state: {
			isLoading,
			showAlertBanner: isError,
		},
		initialState: {
			...defaultOptions.initialState,
			sorting: [{ id: "person_message_id", desc: true }],
		},
	});

	return <SortableTable table={table} title={"Players"} hideHeader />;
};
