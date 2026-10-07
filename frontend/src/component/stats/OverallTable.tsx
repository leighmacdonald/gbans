import Chip from "@mui/material/Chip";
import Stack from "@mui/material/Stack";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { useNavigate } from "@tanstack/react-router";
import { createMRTColumnHelper, type MRT_SortingState, useMaterialReactTable } from "material-react-table";
import { useCallback, useMemo } from "react";
import { renderTableError } from "../../error";
import classDemo from "../../icons/class_demo.png";
import classEngineer from "../../icons/class_engineer.png";
import classHeavy from "../../icons/class_heavy.png";
import classMedic from "../../icons/class_medic.png";
import classPyro from "../../icons/class_pyro.png";
import classScout from "../../icons/class_scout.png";
import classSniper from "../../icons/class_sniper.png";
import classSoldier from "../../icons/class_soldier.png";
import classSpy from "../../icons/class_spy.png";
import { Route } from "../../routes/_auth.match.$matchId";
import { Team } from "../../rpc/stats/v1/stats_pb.ts";
import { blu, red } from "../../theme.ts";
import { PersonCell } from "../PersonCell";
import { createDefaultTableOptions, type OnChangeFn } from "../table/options";
import { SortableTable } from "../table/SortableTable";
import type { MatchRow, MatchView } from "./match";
import { VariantDetailPanel } from "./WeaponDetailPanel";

const overallColumnHelper = createMRTColumnHelper<MatchRow>();
const defaultOverallOptions = createDefaultTableOptions<MatchRow>();
const colSize = 70;

const shortHeader = (short: string, full: string) => ({
	header: short,
	Header: () => (
		<Tooltip title={full}>
			<Typography variant="body2" sx={{ fontWeight: 700 }}>
				{short}
			</Typography>
		</Tooltip>
	),
});

const teamLabel = (team: Team): string => {
	switch (team) {
		case Team.RED:
			return "RED";
		case Team.BLU:
			return "BLU";
		default:
			return "SPEC";
	}
};

const teamColor = (team: Team): string | undefined => {
	switch (team) {
		case Team.RED:
			return red;
		case Team.BLU:
			return blu;
		default:
			return undefined;
	}
};

const rowTint = (team: Team): string | undefined => {
	// Subtle team striping that works in both light and dark mode.
	switch (team) {
		case Team.RED:
			return "rgba(167, 88, 75, 0.14)";
		case Team.BLU:
			return "rgba(84, 125, 140, 0.14)";
		default:
			return undefined;
	}
};

const format1dp = (value: number): string => (Number.isFinite(value) ? value.toFixed(1) : "0.0");

const classIcons: Record<string, string> = {
	demo: classDemo,
	engineer: classEngineer,
	heavy: classHeavy,
	medic: classMedic,
	pyro: classPyro,
	scout: classScout,
	sniper: classSniper,
	soldier: classSoldier,
	spy: classSpy,
};

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
			overallColumnHelper.accessor("team", {
				grow: false,
				...shortHeader("Team", "Team"),
				size: 80,
				Cell: ({ cell }) => {
					const team = cell.getValue<Team>();
					return (
						<Chip
							label={teamLabel(team)}
							size="small"
							sx={{
								backgroundColor: teamColor(team),
								color: "#fff",
								fontWeight: 700,
								minWidth: 52,
							}}
						/>
					);
				},
			}),
			overallColumnHelper.accessor("player", {
				grow: true,
				header: "Player",
				enablePinning: true,
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
			overallColumnHelper.accessor("classes", {
				grow: false,
				...shortHeader("C", "Classes played"),
				size: 110,
				enableSorting: false,
				Cell: ({ cell }) => {
					const classes = cell.getValue<string[]>() ?? [];
					if (classes.length === 0) {
						return <Typography variant="body2">—</Typography>;
					}
					return (
						<Stack direction="row" spacing={0.5} useFlexGap sx={{ flexWrap: "wrap" }}>
							{classes.map((c) => (
								<Tooltip key={c} title={c}>
									<img
										src={classIcons[c.toLowerCase()] ?? ""}
										alt={c}
										loading="lazy"
										style={{ width: 24, height: 24 }}
									/>
								</Tooltip>
							))}
						</Stack>
					);
				},
			}),
			overallColumnHelper.accessor("kills", {
				grow: false,
				...shortHeader("K", "Kills"),
				sortDescFirst: true,
				size: colSize,
			}),
			overallColumnHelper.accessor("assists", {
				grow: false,
				...shortHeader("A", "Assists"),
				sortDescFirst: true,
				size: colSize,
			}),
			overallColumnHelper.accessor("deaths", {
				grow: false,
				...shortHeader("D", "Deaths"),
				sortDescFirst: true,
				size: colSize,
			}),
			overallColumnHelper.accessor("damage", {
				grow: false,
				...shortHeader("DA", "Damage dealt"),
				sortDescFirst: true,
				size: 80,
			}),
			overallColumnHelper.accessor("damagePerMin", {
				grow: false,
				...shortHeader("DA/M", "Damage dealt per minute"),
				sortDescFirst: true,
				size: 80,
				Cell: ({ cell }) => Math.round(cell.getValue<number>()).toLocaleString(),
			}),
			overallColumnHelper.accessor("kaPerD", {
				grow: false,
				...shortHeader("KA/D", "(Kills + Assists) / Deaths"),
				sortDescFirst: true,
				size: colSize,
				Cell: ({ cell }) => format1dp(cell.getValue<number>()),
			}),
			overallColumnHelper.accessor("kd", {
				grow: false,
				...shortHeader("K/D", "Kills / Deaths"),
				sortDescFirst: true,
				size: colSize,
				Cell: ({ cell }) => format1dp(cell.getValue<number>()),
			}),
			overallColumnHelper.accessor("dt", {
				grow: false,
				...shortHeader("DT", "Damage taken"),
				sortDescFirst: true,
				size: 80,
			}),
			overallColumnHelper.accessor("dtPerMin", {
				grow: false,
				...shortHeader("DT/M", "Damage taken per minute"),
				sortDescFirst: true,
				size: 80,
				Cell: ({ cell }) => Math.round(cell.getValue<number>()).toLocaleString(),
			}),
			overallColumnHelper.accessor("healing", {
				grow: false,
				...shortHeader("HP", "Healing"),
				sortDescFirst: true,
				size: 80,
			}),
			overallColumnHelper.accessor("charges", {
				grow: false,
				...shortHeader("UC", "ÜberCharges (all types)"),
				sortDescFirst: true,
				size: colSize,
			}),
			overallColumnHelper.accessor("drops", {
				grow: false,
				...shortHeader("Drops", "Charge drops"),
				sortDescFirst: true,
				size: colSize,
			}),
			overallColumnHelper.accessor("as", {
				grow: false,
				...shortHeader("AS", "Airshots"),
				sortDescFirst: true,
				size: colSize,
			}),
			overallColumnHelper.accessor("bs", {
				grow: false,
				...shortHeader("BS", "Backstabs"),
				sortDescFirst: true,
				size: colSize,
			}),
			overallColumnHelper.accessor("hs", {
				grow: false,
				...shortHeader("HS", "Headshots"),
				sortDescFirst: true,
				size: colSize,
			}),
			overallColumnHelper.accessor("cap", {
				grow: false,
				...shortHeader("CAP", "Point captures"),
				sortDescFirst: true,
				size: colSize,
			}),
			overallColumnHelper.accessor("capturesBlocked", {
				grow: false,
				...shortHeader("Blocked", "Captures blocked"),
				sortDescFirst: true,
				size: 80,
			}),
			overallColumnHelper.accessor("points", {
				grow: false,
				...shortHeader("P", "Scoreboard points"),
				sortDescFirst: true,
				size: colSize,
			}),
		],
		[],
	);

	const overallTable = useMaterialReactTable({
		...defaultOverallOptions,
		columns,
		data: data?.summaries || [],
		enableColumnPinning: true,
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
			columnPinning: {
				left: ["player"],
				right: [],
			},
			columnVisibility: {
				points: false,
			},
		},
		muiToolbarAlertBannerProps: renderTableError(error),
		enableRowActions: false,
		muiTableBodyRowProps: ({ row }) => ({
			sx: {
				backgroundColor: rowTint(row.original.team),
			},
		}),
	});

	return <SortableTable table={overallTable} title={"Overall Match Stats"} hidePagination={true} />;
};
