import Stack from "@mui/material/Stack";
import { useNavigate } from "@tanstack/react-router";
import { createMRTColumnHelper, type MRT_SortingState, useMaterialReactTable } from "material-react-table";
import { useCallback, useMemo } from "react";
import { renderTableError } from "../../error";
import { Route } from "../../routes/_auth.match.$matchId";
import { PersonCell } from "../PersonCell";
import { createDefaultTableOptions, type OnChangeFn } from "../table/options";
import { SortableTable } from "../table/SortableTable";
import type { MatchRow, MatchView } from "./match";
import { VariantDetailPanel } from "./WeaponDetailPanel";

const overallColumnHelper = createMRTColumnHelper<MatchRow>();
const defaultOverallOptions = createDefaultTableOptions<MatchRow>();
const colSize = 100;

export const OverallTable = ({
	data,
	matchId,
	isLoading,
	isError,
	error,
}: {
	data?: MatchView;
	matchId: string;
	isLoading: boolean;
	isError: boolean;
	error: unknown;
}) => {
	const search = Route.useSearch();
	const navigate = useNavigate();
	const setSorting: OnChangeFn<MRT_SortingState> = useCallback(
		(updater) => {
			navigate({
				to: Route.fullPath,
				params: { matchId },
				search: {
					...search,
					sorting: typeof updater === "function" ? updater(search.sorting ?? []) : updater,
				},
			});
		},
		[search, navigate, matchId],
	);

	const columns = useMemo(
		() => [
			overallColumnHelper.accessor("player", {
				grow: false,
				header: "Player",
				sortingFn: (rowA, rowB) => {
					return rowA.original.player.name.toLocaleLowerCase() > rowB.original.player.name.toLocaleLowerCase()
						? -1
						: 1;
				},
				Cell: ({ cell }) => {
					const v = cell.getValue();
					return <PersonCell steamId={v.steamId} avatarHash={v.avatarHash} personaName={v.name} />;
				},
			}),
			// overallColumnHelper.accessor("points", {
			// 	grow: false,
			// 	header: "Pnt",
			// 	sortDescFirst: true,
			// 	size: colSize,
			// }),
			overallColumnHelper.accessor("kills", {
				grow: false,
				header: "K",
				sortDescFirst: true,
				size: colSize,
			}),
			overallColumnHelper.accessor("assists", {
				grow: false,
				header: "A",
				sortDescFirst: true,
				size: colSize,
			}),
			overallColumnHelper.accessor("deaths", {
				grow: false,
				header: "D",
				sortDescFirst: true,
				size: colSize,
			}),
			overallColumnHelper.accessor("healing", {
				grow: false,
				header: "H",
				sortDescFirst: true,
				size: colSize,
			}),
			overallColumnHelper.accessor("damage", {
				grow: false,
				header: "DA",
				size: colSize,
			}),
			overallColumnHelper.accessor("dt", {
				grow: false,
				header: "DT",
				size: colSize,
			}),
			overallColumnHelper.accessor("as", {
				grow: false,
				header: "AS",
				size: colSize,
			}),
			overallColumnHelper.accessor("bs", {
				grow: false,
				header: "BS",
				size: colSize,
			}),
			overallColumnHelper.accessor("cap", {
				grow: false,
				header: "CAP",
			}),
			overallColumnHelper.accessor("capturesBlocked", {
				grow: false,
				header: "Blocked",
			}),
			// overallColumnHelper.accessor("shots", {
			// 	grow: true,
			// 	header: "S/H (%)",
			// 	sortDescFirst: true,
			// 	Cell: ({ row }) => {
			// 		const hitPct = ((row.original.shots / row.original.hits) * 100).toFixed(2);
			// 		return (
			// 			<Typography>
			// 				{row.original.shots}/{row.original.hits} ({hitPct})
			// 			</Typography>
			// 		);
			// 	},
			// }),
		],
		[],
	);

	const overallTable = useMaterialReactTable({
		...defaultOverallOptions,
		columns,
		data: data?.summaries || [],
		enableFilters: false,
		enableFacetedValues: false,
		enableColumnActions: false,
		onSortingChange: setSorting,
		enablePagination: false,
		renderDetailPanel: ({ row }) =>
			data?.summaries ? (
				<Stack>
					<VariantDetailPanel match={data} steamId={row.original.player.steamId} isWeapons={true} />
					<VariantDetailPanel match={data} steamId={row.original.player.steamId} isWeapons={false} />
				</Stack>
			) : null,
		// displayColumnDefOptions: makeRowActionsDefOptions(2),
		state: {
			isLoading,
			showAlertBanner: isError,
			sorting: search.sorting,
		},
		initialState: {
			...defaultOverallOptions.initialState,
			columnVisibility: {
				team: true,
				name: true,
			},
		},
		muiToolbarAlertBannerProps: renderTableError(error),
		enableRowActions: false,
		muiTableBodyRowProps: ({ row }) => ({
			style: {
				backgroundColor: `${row.original.team === "red" ? "#012344" : "#f33333"} !important`, // Conditional color
			},
		}),
	});

	return <SortableTable table={overallTable} title={"Overall Match Stats"} hidePagination={true} />;
};
