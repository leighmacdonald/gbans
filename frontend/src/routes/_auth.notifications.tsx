import LinkIcon from "@mui/icons-material/Link";
import Grid from "@mui/material/Grid";
import { useTheme } from "@mui/material/styles";
import Typography from "@mui/material/Typography";
import { createFileRoute, Link } from "@tanstack/react-router";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useMemo } from "react";
import { PersonCell } from "../component/PersonCell.tsx";
import { BoolCell } from "../component/table/BoolCell.tsx";
import { createDefaultTableOptions } from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { TableCellRelativeDateField } from "../component/table/TableCellRelativeDateField.tsx";
import { TableCellString } from "../component/table/TableCellString.tsx";
import { notifications } from "../rpc/notification/v1/notification-NotificationService_connectquery.ts";
import { useQuery } from "@connectrpc/connect-query";
import { Severity, type UserNotification } from "../rpc/notification/v1/notification_pb.ts";

const columnHelper = createMRTColumnHelper<UserNotification>();
const defaultOptions = createDefaultTableOptions<UserNotification>();

export const Route = createFileRoute("/_auth/notifications")({
	component: NotificationsPage,
	head: ({ match }) => ({
		meta: [{ name: "description", content: "User Notifications" }, match.context.title("Notifications")],
	}),
});

function NotificationsPage() {
	const { data, isLoading, isError } = useQuery(notifications);
	// const queryClient = useQueryClient();
	// const { notifications } = Route.useLoaderData();
	// const { sendError, sendFlash } = useUserFlashCtx();

	// const onMarkAllRead = useMutation({
	// 	mutationKey: ["notifications"],
	// 	mutationFn: async () => {
	// 		await apiNotificationsMarkAllRead();
	// 	},
	// 	onSuccess: () => {
	// 		queryClient.setQueryData(["notifications"], (prev: UserNotification[]) => {
	// 			return prev?.map((n) => {
	// 				return { ...n, read: true };
	// 			});
	// 		});
	// 		sendFlash("success", `Successfully marked ${notifications?.length} as read`);
	// 	},
	// 	onError: sendError,
	// });

	// const onMarkSelected = useMutation({
	// 	mutationKey: ["notifications"],
	// 	mutationFn: async (selectedIds: number[]) => {
	// 		await apiNotificationsMarkRead(selectedIds);
	// 	},
	// 	onSuccess: (_, ids) => {
	// 		queryClient.setQueryData(["notifications"], (prev: UserNotification[]) => {
	// 			return prev?.map((n) => {
	// 				return ids.includes(n.person_notification_id) ? { ...n, read: true } : n;
	// 			});
	// 		});
	// 		sendFlash("success", `Successfully marked ${ids?.length} as read`);
	// 	},
	// 	onError: sendError,
	// });

	// const onDeleteAll = useMutation({
	// 	mutationKey: ["notifications"],
	// 	mutationFn: async () => {
	// 		await apiNotificationsDeleteAll();
	// 	},
	// 	onSuccess: () => {
	// 		queryClient.setQueryData(["notifications"], []);
	// 		sendFlash("success", `Successfully deleted ${notifications?.length} messages`);
	// 	},
	// 	onError: sendError,
	// });

	// const onDeleteSelected = useMutation({
	// 	mutationKey: ["notifications"],
	// 	mutationFn: async (selectedIds: number[]) => {
	// 		await apiNotificationsDelete(selectedIds);
	// 	},
	// 	onSuccess: (_, ids) => {
	// 		queryClient.setQueryData(["notifications"], (prev: UserNotification[]) => {
	// 			return prev?.filter((n) => {
	// 				return !ids.includes(n.person_notification_id);
	// 			});
	// 		});
	// 		sendFlash("success", `Successfully deleted ${ids?.length} messages`);
	// 	},
	// 	onError: sendError,
	// });

	// const onConfirmDeleteSelected = useCallback(async () => {
	// 	const ids = selectedToIds();
	// 	if (ids?.length === 0) {
	// 		return;
	// 	}
	// 	const confirmed = (await NiceModal.show(ConfirmationModal, {
	// 		title: `Delete ${ids.length} notifications?`,
	// 		children: "This cannot be undone",
	// 	})) as boolean;
	// 	if (!confirmed) {
	// 		return;
	// 	}
	// 	onDeleteSelected.mutate(ids);
	// }, [onDeleteSelected, selectedToIds]);

	// const onConfirmDeleteAll = useCallback(async () => {
	// 	if (!notifications) {
	// 		return;
	// 	}
	// 	const confirmed = (await NiceModal.show(ConfirmationModal, {
	// 		title: `Delete all ${notifications.length} notifications?`,
	// 		children: "This cannot be undone",
	// 	})) as boolean;
	// 	if (!confirmed) {
	// 		return;
	// 	}
	// 	onDeleteAll.mutate();
	// }, [notifications, onDeleteAll]);

	// const newMessages = useMemo(() => {
	// 	return notifications?.filter((n) => !n.read).length;
	// }, [notifications]);

	// const buttons = useMemo(() => {
	// 	if (breakMatched) {
	// 		return (
	// 			<ButtonGroup variant="contained" key={"hdr-buttons"}>
	// 				<Button
	// 					startIcon={<DoneIcon />}
	// 					color={"success"}
	// 					key={"mark-selected"}
	// 					onClick={() => {
	// 						const ids = selectedToIds();
	// 						if (ids?.length === 0) {
	// 							return;
	// 						}
	// 						onMarkSelected.mutate(ids);
	// 					}}
	// 					disabled={Object.values(rowSelection).length === 0}
	// 				>
	// 					Mark Selected Read
	// 				</Button>
	// 				<Button
	// 					startIcon={<DoneAllIcon />}
	// 					color={"success"}
	// 					key={"mark-all"}
	// 					onClick={() => onMarkAllRead.mutate()}
	// 					disabled={(notifications ?? [])?.length === 0}
	// 				>
	// 					Mark All Read
	// 				</Button>
	// 				<Button
	// 					startIcon={<RemoveIcon />}
	// 					color={"error"}
	// 					key={"delete-selected"}
	// 					onClick={onConfirmDeleteSelected}
	// 					disabled={Object.values(rowSelection).length === 0}
	// 				>
	// 					Delete Selected
	// 				</Button>
	// 				<Button
	// 					startIcon={<ClearAllIcon />}
	// 					color={"error"}
	// 					key={"delete-all"}
	// 					onClick={onConfirmDeleteAll}
	// 					disabled={(notifications ?? [])?.length === 0}
	// 				>
	// 					Delete All
	// 				</Button>
	// 			</ButtonGroup>
	// 		);
	// 	} else {
	// 		return (
	// 			<ButtonGroup variant="contained" key={"hdr-buttons"}>
	// 				<Tooltip title="Mark Selected Read">
	// 					<IconButton
	// 						color={"success"}
	// 						key={"mark-selected"}
	// 						onClick={() => {
	// 							const ids = selectedToIds();
	// 							if (ids?.length === 0) {
	// 								return;
	// 							}
	// 							onMarkSelected.mutate(ids);
	// 						}}
	// 						disabled={Object.values(rowSelection).length === 0}
	// 					>
	// 						<DoneIcon />
	// 					</IconButton>
	// 				</Tooltip>
	// 				<Tooltip title="Mark All Read">
	// 					<IconButton
	// 						color={"success"}
	// 						key={"mark-all"}
	// 						onClick={() => onMarkAllRead.mutate()}
	// 						disabled={(notifications ?? [])?.length === 0}
	// 					>
	// 						<DoneAllIcon />
	// 					</IconButton>
	// 				</Tooltip>
	// 				<Tooltip title="Delete Selected">
	// 					<IconButton
	// 						color={"error"}
	// 						key={"delete-selected"}
	// 						onClick={onConfirmDeleteSelected}
	// 						disabled={Object.values(rowSelection).length === 0}
	// 					>
	// 						<RemoveIcon />
	// 					</IconButton>
	// 				</Tooltip>
	// 				<Tooltip title="Delete All">
	// 					<IconButton
	// 						color={"error"}
	// 						key={"delete-all"}
	// 						onClick={onConfirmDeleteAll}
	// 						disabled={(notifications ?? [])?.length === 0}
	// 					>
	// 						<ClearAllIcon />
	// 					</IconButton>
	// 				</Tooltip>
	// 			</ButtonGroup>
	// 		);
	// 	}
	// }, [
	// 	breakMatched,
	// 	onConfirmDeleteAll,
	// 	onConfirmDeleteSelected,
	// 	rowSelection,
	// 	notifications,
	// 	onMarkAllRead.mutate,
	// 	onMarkSelected.mutate,
	// 	selectedToIds,
	// ]);

	const columns = useMemo(() => {
		return [
			columnHelper.accessor("read", {
				header: "Read",
				grow: false,
				Cell: ({ cell }) => <BoolCell enabled={Boolean(cell.getValue())} />,
			}),
			columnHelper.accessor("createdOn", {
				header: "Created",
				grow: false,
				Cell: ({ cell }) => <TableCellRelativeDateField date={cell.getValue()} suffix={true} />,
			}),
			columnHelper.accessor("severity", {
				header: "level",
				grow: false,
				Cell: ({ cell }) => <TableCellSeverity severity={cell.getValue()} />,
			}),

			columnHelper.accessor("message", {
				header: "Message",
				grow: true,
				Cell: ({ cell }) => <TableCellString>{cell.getValue() as string}</TableCellString>,
			}),

			columnHelper.accessor("author", {
				Cell: (info) =>
					info.row.original.author != null ? (
						<PersonCell
							steam_id={info.row.original.author.steam_id}
							personaname={info.row.original.author?.name}
							avatar_hash={info.row.original.author?.avatarhash}
						/>
					) : (
						""
					),
				header: "Author",
			}),
			columnHelper.accessor("link", {
				header: "Link",
				grow: false,
				Cell: ({ cell }) => {
					return cell.getValue() ? (
						<Link to={cell.getValue() as string}>
							<LinkIcon color={"primary"} />
						</Link>
					) : (
						""
					);
				},
			}),
		];
	}, []);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data?.notifications ?? [],
		enableFilters: true,
		enableHiding: true,
		enableFacetedValues: true,
		state: {
			isLoading,
			showAlertBanner: isError,
		},
		initialState: {
			...defaultOptions.initialState,
			sorting: [{ id: "ban_id", desc: true }],
			columnVisibility: {
				cidr_block_whitelist_id: false,
				address: true,
				created_on: true,
				updated_on: false,
			},
		},
		enableRowActions: true,
		renderRowActionMenuItems: () => [],
	});

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<SortableTable table={table} title={"User Notifications"} />
			</Grid>
		</Grid>
	);
}

const TableCellSeverity = ({ severity }: { severity: Severity }) => {
	const theme = useTheme();

	switch (severity) {
		case Severity.ERROR:
			return <Typography style={{ color: theme.palette.error.main }}>ERROR</Typography>;
		case Severity.WARN:
			return <Typography style={{ color: theme.palette.warning.main }}>WARN</Typography>;
		default:
			return <Typography style={{ color: theme.palette.info.main }}>INFO</Typography>;
	}
};
