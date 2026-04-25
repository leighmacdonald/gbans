import { timestampDate } from "@bufbuild/protobuf/wkt";
import { useQuery } from "@connectrpc/connect-query";
import NiceModal from "@ebay/nice-modal-react";
import VpnKeyIcon from "@mui/icons-material/VpnKey";
import { useTheme } from "@mui/material";
import Grid from "@mui/material/Grid";
import IconButton from "@mui/material/IconButton";
import Typography from "@mui/material/Typography";
import { createFileRoute, stripSearchParams, useNavigate } from "@tanstack/react-router";
import {
	createMRTColumnHelper,
	type MRT_ColumnFiltersState,
	type MRT_PaginationState,
	type MRT_SortingState,
	useMaterialReactTable,
} from "material-react-table";
import { useCallback, useMemo } from "react";
import { PersonEditModal } from "../component/modal/PersonEditModal.tsx";
import { PersonCell } from "../component/PersonCell.tsx";
import RouterLink from "../component/RouterLink.tsx";
import { RowActionContainer } from "../component/RowActionContainer.tsx";
import { BoolCell } from "../component/table/BoolCell.tsx";
import {
	createDefaultTableOptions,
	filterValue,
	filterValueBool,
	filterValueNumber,
	filterValueNumberArray,
	makeSchemaDefaults,
	makeSchemaState,
	type OnChangeFn,
	setColumnFilter,
	sortValueDefault,
} from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { TableCellRelativeDateField } from "../component/table/TableCellRelativeDateField.tsx";
import { useAuth } from "../hooks/useAuth.ts";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import { type Person, VisibilityState } from "../rpc/person/v1/person_pb.ts";
import { query } from "../rpc/person/v1/person-PersonService_connectquery.ts";
import { Privilege } from "../rpc/person/v1/privilege_pb.ts";
import { enumValues } from "../util/lists.ts";

const defaultValues = makeSchemaDefaults({ defaultColumn: "created_on" });
const validateSearch = makeSchemaState("created_on");
const columnHelper = createMRTColumnHelper<Person>();
const defaultOptions = createDefaultTableOptions<Person>();

export const Route = createFileRoute("/_mod/admin/people")({
	component: AdminPeople,
	validateSearch,
	search: {
		middlewares: [stripSearchParams(defaultValues)],
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "People" }, match.context.title("People")],
	}),
});

function AdminPeople() {
	const search = Route.useSearch();
	const navigate = useNavigate();
	const theme = useTheme();
	const { sendFlash } = useUserFlashCtx();
	const { hasPermission } = useAuth();

	const sort = search.sorting ? sortValueDefault(search.sorting, "created_on") : undefined;
	const steam_id = filterValue("steam_id", search.columnFilters);

	const { data, isLoading, isError, isRefetching } = useQuery(query, {
		filter: {
			desc: sort ? sort.desc : true,
			limit: BigInt(search.pagination?.pageSize ?? 25n),
			offset: BigInt(search.pagination ? search.pagination.pageIndex * search.pagination?.pageSize : 0n),
			orderBy: sort ? sort.id : "created_on",
		},
		steamIds: steam_id && steam_id !== "" ? [steam_id] : [],
		vacBans: filterValueNumber("vac_bans", search.columnFilters),
		gameBans: filterValueNumber("game_bans", search.columnFilters),
		communityBanned: filterValueBool("community_banned", search.columnFilters),
		withPermissions: filterValueNumberArray<Person, Privilege>("permissionLevel", search.columnFilters),
	});

	const onEditPerson = useCallback(
		async (person: Person) => {
			try {
				await NiceModal.show(PersonEditModal, {
					person,
				});
				sendFlash("success", "Updated permission level successfully");
			} catch (e) {
				sendFlash("error", `${e}`);
			}
		},
		[sendFlash],
	);

	const setSorting: OnChangeFn<MRT_SortingState> = useCallback(
		async (updater) => {
			await navigate({
				to: Route.fullPath,
				search: {
					...search,
					sorting: typeof updater === "function" ? updater(search.sorting ?? []) : updater,
				},
			});
		},
		[search, navigate],
	);

	const setColumnFilters: OnChangeFn<MRT_ColumnFiltersState> = useCallback(
		async (updater) => {
			await navigate({
				to: Route.fullPath,
				search: {
					...search,
					columnFilters: typeof updater === "function" ? updater(search.columnFilters ?? []) : updater,
				},
			});
		},
		[search, navigate],
	);

	const setPagination: OnChangeFn<MRT_PaginationState> = useCallback(
		async (updater) => {
			await navigate({
				to: Route.fullPath,
				search: {
					...search,
					pagination: search.pagination
						? typeof updater === "function"
							? updater(search.pagination)
							: updater
						: undefined,
				},
			});
		},
		[search, navigate],
	);

	const columns = useMemo(() => {
		return [
			columnHelper.accessor("steamId", {
				header: "SteamID",
				grow: true,
				Cell: ({ row }) => {
					return (
						<PersonCell
							steam_id={row.original.steamId}
							personaname={row.original.personaName}
							avatar_hash={row.original.avatarHash}
						>
							<RouterLink
								style={{
									color:
										theme.palette.mode === "dark"
											? theme.palette.primary.light
											: theme.palette.primary.dark,
								}}
								to={Route.fullPath}
								search={setColumnFilter(search, "steam_id", row.original.steamId)}
							>
								{row.original.personaName ?? row.original.steamId}
							</RouterLink>
						</PersonCell>
					);
				},
			}),
			columnHelper.accessor("visibilityState", {
				header: "Visibility",
				grow: false,
				Cell: ({ cell }) => (
					<Typography variant={"body1"}>
						{cell.getValue() === VisibilityState.PUBLIC ? "Public" : "Private"}
					</Typography>
				),
			}),
			columnHelper.accessor("vacBans", {
				header: "Vac Bans",
				grow: false,
				Cell: ({ cell }) => (
					<Typography variant={"body1"}>{cell.getValue() > 0 ? cell.getValue() : ""}</Typography>
				),
			}),
			columnHelper.accessor("communityBanned", {
				header: "Comm Ban",
				grow: false,
				filterVariant: "checkbox",
				Cell: ({ cell }) => <BoolCell enabled={cell.getValue()} />,
			}),

			columnHelper.accessor("timeCreated", {
				header: "Created",
				grow: false,
				Cell: ({ cell }) => {
					const value = cell.getValue();
					if (!value) {
						return;
					}

					return <TableCellRelativeDateField date={timestampDate(value)} />;
				},
			}),

			columnHelper.accessor("createdOn", {
				header: "First Seen",
				grow: false,
				Cell: ({ cell }) => {
					const value = cell.getValue();
					if (!value) {
						return;
					}

					return <TableCellRelativeDateField date={timestampDate(value)} />;
				},
			}),

			columnHelper.accessor("permissionLevel", {
				header: "Perms",
				grow: false,
				filterVariant: "multi-select",
				filterSelectOptions: enumValues(Privilege).map((perm) => ({
					label: Privilege[perm],
					value: perm,
				})),
				Cell: ({ row }) => (
					<Typography>{Privilege[row.original ? row.original.permissionLevel : Privilege.GUEST]}</Typography>
				),
			}),
		];
	}, [theme, search]);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data ? data.people : [],
		rowCount: Number(data ? data.count : 0),
		enableFilters: true,
		enableRowActions: true,
		state: {
			columnFilters: search.columnFilters,
			isLoading: isLoading || isRefetching,
			pagination: search.pagination,
			showAlertBanner: isError,
			showProgressBars: isRefetching,
			sorting: search.sorting,
		},
		initialState: {
			...defaultOptions.initialState,
			columnVisibility: {
				steam_id: true,
				source_id: true,
				body: true,
				created_on: true,
			},
		},
		manualFiltering: true,
		manualPagination: true,
		manualSorting: true,
		onColumnFiltersChange: setColumnFilters,
		onPaginationChange: setPagination,
		onSortingChange: setSorting,
		renderRowActions: ({ row }) => (
			<RowActionContainer>
				<IconButton
					disabled={!hasPermission(Privilege.ADMIN)}
					color={"warning"}
					onClick={() => onEditPerson(row.original)}
					key={"editperms"}
				>
					<VpnKeyIcon />
				</IconButton>
			</RowActionContainer>
		),
	});
	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<SortableTable table={table} title={"Player Search"} />
			</Grid>
		</Grid>
	);
}
