import { useMutation, useQuery } from "@connectrpc/connect-query";
import NiceModal from "@ebay/nice-modal-react";
import AddIcon from "@mui/icons-material/Add";
import DeleteIcon from "@mui/icons-material/Delete";
import EditIcon from "@mui/icons-material/Edit";
import InfoIcon from "@mui/icons-material/Info";
import Grid from "@mui/material/Grid";
import IconButton from "@mui/material/IconButton";
import TableCell from "@mui/material/TableCell";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { createFileRoute } from "@tanstack/react-router";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useCallback, useMemo } from "react";
import { ContainerWithHeader } from "../component/ContainerWithHeader.tsx";
import { ConfirmationModal } from "../component/modal/ConfirmationModal.tsx";
import { FilterEditModal } from "../component/modal/FilterEditModal.tsx";
import { PersonCell } from "../component/PersonCell.tsx";
import { RowActionContainer } from "../component/RowActionContainer.tsx";
import { BoolCell } from "../component/table/BoolCell.tsx";
import { createDefaultTableOptions, makeRowActionsDefOptions } from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { TableCellSmall } from "../component/table/TableCellSmall.tsx";
import { TableCellString } from "../component/table/TableCellString.tsx";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import { BanType } from "../rpc/ban/v1/ban_pb.ts";
import { type Filter, FilterAction, type UserWarning } from "../rpc/chat/v1/wordfilter_pb.ts";
import { filterDelete, filters, warningState } from "../rpc/chat/v1/wordfilter-WordfilterService_connectquery.ts";
import { renderTimestamp } from "../util/time.ts";

const columnHelper = createMRTColumnHelper<Filter>();
const defaultOptions = createDefaultTableOptions<Filter>();

export const Route = createFileRoute("/_mod/admin/filters")({
	component: AdminFilters,
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Filtered Words" }, match.context.title("Filtered Words")],
	}),
});

function AdminFilters() {
	const { queryClient } = Route.useRouteContext();
	const { sendFlash, sendError } = useUserFlashCtx();

	const { data, isLoading, isError } = useQuery(filters);

	const onCreate = useCallback(async () => {
		try {
			const resp = (await NiceModal.show(FilterEditModal, {})) as Filter;
			queryClient.setQueryData(["filters"], [...(data?.filters ?? []), resp]);
		} catch (e) {
			sendFlash("error", `${e}`);
		}
	}, [queryClient, sendFlash, data]);

	const onEdit = useCallback(
		async (filter: Filter) => {
			try {
				const resp = (await NiceModal.show(FilterEditModal, {
					filter,
				})) as Filter;

				queryClient.setQueryData(
					["filters"],
					(data?.filters ?? []).map((f) => {
						return f.filterId === resp.filterId ? resp : f;
					}),
				);
			} catch (e) {
				sendFlash("error", `${e}`);
			}
		},
		[data, queryClient, sendFlash],
	);

	const deleteMutation = useMutation(filterDelete, {
		onSuccess: (_, filterId) => {
			sendFlash("success", `Deleted filter: ${filterId}`);
		},
		onError: sendError,
	});

	const onDelete = useCallback(
		async (filter: Filter) => {
			try {
				const confirmed = (await NiceModal.show(ConfirmationModal, {
					title: `Are you sure you want to delete this filter?`,
				})) as boolean;

				if (!confirmed || !filter.filterId) {
					return;
				}
				await deleteMutation.mutateAsync({ filterId: filter.filterId });

				queryClient.setQueryData(
					["filters"],
					(data?.filters ?? []).filter((f) => f.filterId !== filter.filterId),
				);
			} catch (e) {
				sendFlash("success", `${e}`);
				return;
			}
		},
		[deleteMutation, data, queryClient, sendFlash],
	);

	const columns = useMemo(() => {
		return [
			columnHelper.accessor("pattern", {
				header: "Pattern",
				grow: true,
				minSize: 350,
				enableColumnFilter: true,
				meta: {
					tooltip: "Find and patterns that match this word or phrase",
				},
				filterFn: (row, _, filterValue) => {
					if (row.original.isRegex) {
						const rx = new RegExp(row.original.pattern);
						return Boolean(rx.exec(filterValue.toLowerCase()));
					}
					return row.original.pattern.toLowerCase().includes(filterValue.toLowerCase());
				},
			}),
			columnHelper.accessor("isRegex", {
				header: "Rx",
				filterVariant: "checkbox",
				enableColumnFilter: false,
				grow: false,
				Cell: ({ cell }) => <BoolCell enabled={cell.getValue()} />,
			}),
			columnHelper.accessor("action", {
				header: "Action",
				filterVariant: "multi-select",
				meta: { tooltip: "What action to take?" },
				grow: false,
				filterSelectOptions: [
					{ label: "Mute", value: BanType.NO_COMM },
					{ label: "Ban", value: BanType.BANNED },
				],
				Cell: ({ cell }) => FilterAction[cell.getValue()],
			}),
			columnHelper.accessor("duration", {
				header: "Duration",
				enableColumnFilter: false,
				grow: false,
				meta: { tooltip: "Duration of the punishment when triggered" },
			}),
			columnHelper.accessor("weight", {
				grow: false,
				enableColumnFilter: false,
				header: "Weight",
			}),
			columnHelper.accessor("triggerCount", {
				header: "Trig #",
				enableColumnFilter: false,
				grow: false,
				meta: { tooltip: "Number of times the filter has been triggered" },
			}),
		];
	}, []);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data?.filters ?? [],
		enableFilters: true,
		enableFacetedValues: true,
		state: {
			isLoading,
			showAlertBanner: isError,
		},
		displayColumnDefOptions: makeRowActionsDefOptions(2),
		enableRowActions: true,
		renderRowActions: ({ row }) => (
			<RowActionContainer>
				<IconButton
					key={"delete"}
					color={"error"}
					onClick={async () => {
						await onDelete(row.original);
					}}
				>
					<DeleteIcon />
				</IconButton>
				<IconButton
					key={"edit"}
					color={"warning"}
					onClick={async () => {
						await onEdit(row.original);
					}}
				>
					<EditIcon />
				</IconButton>
			</RowActionContainer>
		),
		initialState: {
			...defaultOptions.initialState,
			sorting: [{ id: "trigger_count", desc: true }],
			columnVisibility: {
				source_id: false,
				target_id: true,
				reason: true,
				reason_text: true,
				created_on: false,
				updated_on: true,
			},
		},
	});

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<SortableTable
					table={table}
					title={"Word Filters"}
					buttons={[
						<Tooltip title="Create new filter" key="1">
							<IconButton key={`ban-steam`} onClick={onCreate} sx={{ color: "primary.contrastText" }}>
								<AddIcon />
							</IconButton>
						</Tooltip>,
					]}
				/>
			</Grid>
			<Grid size={{ xs: 12 }}>
				<WarningStateTable />
			</Grid>
			<Grid size={{ xs: 12 }}>
				<ContainerWithHeader title={"How it works"} iconLeft={<InfoIcon />}>
					<Typography variant={"body1"}>
						The way the warning tracking works is that each time a user triggers a match, it gets an entry
						in the table based on the weight of the match. The individual match weight is determined by the
						word filter defined above. Once the sum of their triggers exceeds the max weight the user will
						have action taken against them automatically. Matched entries are ephemeral and are removed over
						time based on the configured timeout value.
					</Typography>
				</ContainerWithHeader>
			</Grid>
		</Grid>
	);
}

const columnHelperWarn = createMRTColumnHelper<UserWarning>();
const defaultOptionsWarn = createDefaultTableOptions<UserWarning>();

export const WarningStateTable = () => {
	const { data, isLoading, isError } = useQuery(warningState);

	const renderFilter = useCallback((f: Filter) => {
		const pat = f.isRegex ? (f.pattern as string) : (f.pattern as string);

		return (
			<>
				<Typography variant={"h6"}>Matched {f.isRegex ? "Regex" : "Text"}</Typography>
				<Typography variant={"body1"}>{pat}</Typography>
				<Typography variant={"body1"}>Weight: {f.weight}</Typography>
				<Typography variant={"body1"}>Action: {FilterAction[f.action]}</Typography>
			</>
		);
	}, []);

	const columns = useMemo(
		() => [
			columnHelperWarn.accessor("steamId", {
				header: "Pattern",
				Cell: ({ row }) => (
					<TableCellSmall>
						<PersonCell
							steamId={row.original.steamId}
							personaName={row.original.personaName}
							avatarHash={row.original.avatarHash}
						/>
					</TableCellSmall>
				),
			}),
			columnHelperWarn.accessor("createdOn", {
				header: "Created",
				Cell: ({ cell }) => renderTimestamp(cell.getValue()),
			}),
			columnHelperWarn.accessor("filter.action", {
				header: "Action",
				Cell: ({ cell }) => (
					<TableCellSmall>
						<Typography>
							{typeof cell.getValue() === "undefined" ? "" : FilterAction[cell.getValue()]}
						</Typography>
					</TableCellSmall>
				),
			}),
			columnHelperWarn.accessor("matched", {
				header: "Duration",
				Cell: ({ row, cell }) => (
					<TableCell>
						<Tooltip title={renderFilter(row.original as unknown as Filter)}>
							<Typography>{cell.getValue()}</Typography>
						</Tooltip>
					</TableCell>
				),
			}),
			columnHelperWarn.accessor("currentTotal", {
				header: "Weight",
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			columnHelperWarn.accessor("message", {
				header: "Triggered",
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
		],
		[renderFilter],
	);

	const table = useMaterialReactTable({
		...defaultOptionsWarn,
		columns,
		data: data ? data.current : [],
		enableFilters: true,
		state: {
			isLoading,
			showAlertBanner: isError,
		},
		initialState: {
			...defaultOptionsWarn.initialState,
			sorting: [{ id: "updated_on", desc: true }],
			columnVisibility: {
				source_id: false,
				target_id: true,
				reason: true,
				reason_text: true,
				created_on: false,
				updated_on: true,
			},
		},
	});

	return <SortableTable table={table} title={`Current Warning State (Max Weight: ${data?.maxWeight ?? "..."})`} />;
};
