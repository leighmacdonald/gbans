import Typography from "@mui/material/Typography";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useMemo } from "react";
import { shortHeader } from "../table/columnHeaders";
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
				...shortHeader(isWeapons ? "Weapon" : "Class", isWeapons ? "Weapon used" : "Class played"),
			}),

			roundColumnHelper.accessor("kills", {
				...shortHeader("Kills", "Kills"),
				sortDescFirst: true,
				size: colSize,
			}),

			roundColumnHelper.accessor("assists", {
				...shortHeader("Assists", "Assists"),
				sortDescFirst: true,
				size: colSize,
			}),
			roundColumnHelper.accessor("deaths", {
				...shortHeader("Deaths", "Deaths"),
				sortDescFirst: true,
				size: colSize,
			}),
			roundColumnHelper.accessor("healing", {
				...shortHeader("Healing", "Healing"),
				sortDescFirst: true,
				size: colSize,
			}),
			roundColumnHelper.accessor("damage", {
				...shortHeader("Damage", "Damage dealt"),
				sortDescFirst: true,
				size: colSize,
			}),
			roundColumnHelper.accessor("damageTaken", {
				...shortHeader("Damage Taken", "Damage taken"),
				sortDescFirst: true,
				size: colSize,
			}),
			roundColumnHelper.accessor("airshots", {
				...shortHeader("AS", "Airshots"),
				sortDescFirst: true,
				size: colSize,
			}),
			roundColumnHelper.accessor("headshots", {
				...shortHeader("HS (K)", "Headshots"),
				sortDescFirst: true,
				size: colSize,
			}),
			roundColumnHelper.accessor("headshotKills", {
				...shortHeader("HSK", "Headshot kills"),
				sortDescFirst: true,
				size: colSize,
			}),
			roundColumnHelper.accessor("backstabs", {
				grow: false,
				...shortHeader("BS (K)", "Backstabs"),
				sortDescFirst: true,
				size: colSize,
			}),
			roundColumnHelper.accessor("backstabKills", {
				...shortHeader("BSK", "Backstab kills"),
				sortDescFirst: true,
				size: colSize,
			}),
			roundColumnHelper.accessor("dominations", {
				...shortHeader("DOM", "Dominations"),
				sortDescFirst: true,
				size: colSize,
			}),
			roundColumnHelper.accessor("dominated", {
				...shortHeader("DOMD", "Times dominated"),
				sortDescFirst: true,
				size: colSize,
			}),
			roundColumnHelper.accessor("revenges", {
				...shortHeader("Revenges", "Revenges"),
				sortDescFirst: true,
				size: colSize,
			}),
			roundColumnHelper.accessor("revenged", {
				...shortHeader("Revenged", "Times revenged"),
				sortDescFirst: true,
				size: colSize,
			}),
			roundColumnHelper.accessor("drops", {
				...shortHeader("Drops", "Charge drops"),
				sortDescFirst: true,
				size: colSize,
			}),
			roundColumnHelper.accessor("nearFullChargeDeath", {
				...shortHeader("NFCD", "Deaths with near-full ÜberCharge"),
				sortDescFirst: true,
				size: colSize,
			}),
			roundColumnHelper.accessor("chargesUber", {
				...shortHeader("Uber", "ÜberCharges"),
				sortDescFirst: true,
				size: colSize,
			}),
			roundColumnHelper.accessor("chargesKritz", {
				...shortHeader("Kritz", "Kritzkrieg charges"),
				sortDescFirst: true,
				size: colSize,
			}),
			roundColumnHelper.accessor("chargesVacc", {
				...shortHeader("Vacc", "Vaccinator charges"),
				sortDescFirst: true,
				size: colSize,
			}),
			roundColumnHelper.accessor("chargesQuickfix", {
				...shortHeader("Quickfix", "Quick-Fix charges"),
				sortDescFirst: true,
				size: colSize,
			}),
		],
		[roundColumnHelper, isWeapons],
	);

	const table = useMaterialReactTable({
		...defaultRoundOptions,
		// Keep natural column widths and horizontal-scroll on overflow instead of
		// stretching columns across the full detail panel width.
		layoutMode: "grid-no-grow",
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
