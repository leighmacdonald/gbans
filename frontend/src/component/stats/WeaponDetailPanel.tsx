import { Typography } from "@mui/material";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useMemo } from "react";
import { createDefaultTableOptions } from "../table/options";
import { SortableTable } from "../table/SortableTable";
import type { MatchPlayerVariantStats, MatchView } from "./match";

const colSize = 75;

export const VariantDetailPanel = ({
	match,
	steamId,
	isWeapons,
}: {
	match: MatchView;
	steamId: string;
	isWeapons: boolean;
}) => {
	const roundColumnHelper = createMRTColumnHelper<MatchPlayerVariantStats>();
	const defaultRoundOptions = createDefaultTableOptions<MatchPlayerVariantStats>();

	const data = useMemo(() => {
		const playerWeapons = match.variants[steamId];
		if (!playerWeapons) {
			return [];
		}
		return (
			Object.values(playerWeapons)
				// Skip spammy entries with little/no data.
				// .filter((w) => w.kills > 0 || w.assists > 0 || w.healing > 0 || w.deaths > 0)
				.filter((w) => w.damage > 0)
				.filter((w) => (isWeapons ? w.isWeapon : !w.isWeapon))
		);
	}, [match, steamId, isWeapons]);

	const columns = useMemo(
		() => [
			roundColumnHelper.accessor("name", {
				header: "Player",
			}),

			roundColumnHelper.accessor("kills", {
				header: "Kills",
				sortDescFirst: true,
				size: colSize,
			}),

			roundColumnHelper.accessor("assists", {
				header: "Assists",
				sortDescFirst: true,
				size: colSize,
			}),
			roundColumnHelper.accessor("deaths", {
				header: "Deaths",
				sortDescFirst: true,
				size: colSize,
			}),
			roundColumnHelper.accessor("healing", {
				header: "Healing",
				sortDescFirst: true,
				size: colSize,
			}),
			roundColumnHelper.accessor("damage", {
				header: "Damage",
				sortDescFirst: true,
				size: colSize,
			}),
			roundColumnHelper.accessor("damageTaken", {
				header: "Damage Taken",
				sortDescFirst: true,
				size: colSize,
			}),
			roundColumnHelper.accessor("airshots", {
				header: "AS",
				sortDescFirst: true,
				size: colSize,
			}),
			roundColumnHelper.accessor("headshots", {
				header: "HS (K)",
				sortDescFirst: true,
				size: colSize,
			}),
			roundColumnHelper.accessor("backstabs", {
				grow: false,
				header: "BS (K)",
				sortDescFirst: true,
				size: colSize,
			}),
		],
		[roundColumnHelper],
	);

	const table = useMaterialReactTable({
		...defaultRoundOptions,
		columns,
		data,
		enableFilters: false,
		enableFacetedValues: false,
		enableColumnActions: false,
		enablePagination: false,
		initialState: {
			...defaultRoundOptions.initialState,
			columnVisibility: {
				winner: true,
			},
		},
	});

	if (!data) {
		return <Typography>No shooty?</Typography>;
	}

	return (
		<SortableTable
			table={table}
			title={isWeapons ? "Player Weapons" : "Player Classes"}
			hidePagination={true}
			hideHeader={true}
		/>
	);
};
