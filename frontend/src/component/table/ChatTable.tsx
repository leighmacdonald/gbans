import FlagIcon from "@mui/icons-material/Flag";
import ReportIcon from "@mui/icons-material/Report";
import Button from "@mui/material/Button";
import { useTheme } from "@mui/material/styles";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { useNavigate } from "@tanstack/react-router";
import {
	createColumnHelper,
	getCoreRowModel,
	getPaginationRowModel,
	type OnChangeFn,
	type PaginationState,
	useReactTable,
} from "@tanstack/react-table";
import { useMemo } from "react";
import type { PersonMessage } from "../../schema/people.ts";
import { stringToColour } from "../../util/colours.ts";
import { IconButtonLink } from "../IconButtonLink.tsx";
import { PersonCell } from "../PersonCell.tsx";
import { DataTable } from "./DataTable.tsx";
import { TableCellRelativeDateField } from "./TableCellRelativeDateField.tsx";

const columnHelper = createColumnHelper<PersonMessage>();

export const ChatTable = ({
	messages,
	isLoading,
	manualPaging = true,
	pagination,
	setPagination,
}: {
	messages: PersonMessage[];
	isLoading: boolean;
	manualPaging?: boolean;
	pagination?: PaginationState;
	setPagination?: OnChangeFn<PaginationState>;
}) => {
	const navigate = useNavigate({ from: "/chatlogs" });
	const theme = useTheme();

	const columns = useMemo(() => {
		return [
			columnHelper.accessor("server_id", {
				header: "Server",
				size: 40,
				cell: (info) => (
					<Button
						sx={{
							color: stringToColour(messages[info.row.index].server_name, theme.palette.mode),
						}}
						onClick={async () => {
							await navigate({
								to: "/chatlogs",
								search: (prev) => ({
									...prev,
									server_id: info.getValue(),
								}),
							});
						}}
					>
						{messages[info.row.index].server_name}
					</Button>
				),
			}),

			columnHelper.accessor("created_on", {
				header: "Created",
				size: 80,
				cell: (info) => <TableCellRelativeDateField date={info.row.original.created_on} />,
			}),

			columnHelper.accessor("persona_name", {
				header: "Name",
				cell: (info) => (
					<PersonCell
						showCopy={true}
						steam_id={messages[info.row.index].steam_id}
						avatar_hash={messages[info.row.index].avatar_hash}
						personaname={messages[info.row.index].persona_name}
					/>
				),
			}),

			columnHelper.accessor("body", {
				header: "Message",
				size: 400,
				cell: (info) => (
					<Typography padding={0} variant={"body1"}>
						{info.getValue() as string}
					</Typography>
				),
			}),
			columnHelper.display({
				header: "Flg",
				size: 30,
				cell: (info) =>
					info.row.original.auto_filter_flagged > 0 ? (
						<Tooltip title={"Message already flagged"}>
							<FlagIcon color={"error"} />
						</Tooltip>
					) : null,
			}),
			columnHelper.display({
				header: "Rep",
				size: 30,
				cell: (info) => (
					<Tooltip title={"Create Report"}>
						<IconButtonLink
							color={"error"}
							disabled={info.row.original.auto_filter_flagged > 0}
							to={"/report"}
							search={{
								person_message_id: info.row.original.person_message_id,
								steam_id: info.row.original.steam_id,
							}}
						>
							<ReportIcon />
						</IconButtonLink>
					</Tooltip>
				),
			}),
		];
	}, [messages, navigate, theme.palette.mode]);

	const table = useReactTable({
		data: messages,
		columns: columns,
		getCoreRowModel: getCoreRowModel(),
		manualPagination: manualPaging,
		autoResetPageIndex: true,
		...(manualPaging
			? {}
			: {
					manualPagination: false,
					onPaginationChange: setPagination,
					getPaginationRowModel: getPaginationRowModel(),
					state: { pagination },
				}),
	});

	return <DataTable table={table} isLoading={isLoading} />;
};
