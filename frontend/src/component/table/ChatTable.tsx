import Button from "@mui/material/Button";
import { useTheme } from "@mui/material/styles";
import Typography from "@mui/material/Typography";
import { useQuery } from "@tanstack/react-query";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useMemo } from "react";
import { apiGetMessages } from "../../api/profile.ts";
import type { PersonMessage } from "../../schema/people.ts";
import { stringToColour } from "../../util/colours.ts";
import { PersonCell } from "../PersonCell.tsx";
import { createDefaultTableOptions } from "./options.ts";
import { SortableTable } from "./SortableTable.tsx";
import { TableCellRelativeDateField } from "./TableCellRelativeDateField.tsx";

const columnHelper = createMRTColumnHelper<PersonMessage>();
const defaultOptions = createDefaultTableOptions<PersonMessage>();

export const ChatTable = ({ steamId }: { steamId: string }) => {
	const { data, isLoading, isError } = useQuery({
		queryKey: ["reportChat", { steamId }],
		queryFn: async () => {
			return await apiGetMessages({
				personaname: "",
				query: "",
				source_id: steamId,
				limit: 2500,
				offset: 0,
				order_by: "person_message_id",
				desc: true,
				flagged_only: false,
			});
		},
	});
	const theme = useTheme();
	const columns = useMemo(
		() => [
			columnHelper.accessor("server_id", {
				header: "Server",
				grow: false,
				Cell: ({ row }) => (
					<Button
						variant="text"
						sx={{
							color: stringToColour(row.original.server_name, theme.palette.mode),
						}}
					>
						{row.original.server_name}
					</Button>
				),
			}),

			columnHelper.accessor("created_on", {
				header: "Created",
				grow: false,
				Cell: ({ cell }) => <TableCellRelativeDateField date={cell.getValue()} />,
			}),

			columnHelper.accessor("persona_name", {
				header: "Name",
				grow: false,
				Cell: ({ row }) => (
					<PersonCell
						showCopy={true}
						steam_id={row.original.steam_id}
						avatar_hash={row.original.avatar_hash}
						personaname={row.original.persona_name}
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
		[theme.palette.mode],
	);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data?.data ?? [],
		rowCount: data?.count ?? 0,
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
