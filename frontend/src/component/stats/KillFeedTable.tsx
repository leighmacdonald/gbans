import GavelIcon from "@mui/icons-material/Gavel";
import Typography from "@mui/material/Typography";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useMemo } from "react";
import type { PersonDisplay } from "../../rpc/person/v1/person_core_pb";
import { PersonCell } from "../PersonCell";
import { shortHeader } from "../table/columnHeaders";
import { createDefaultTableOptions } from "../table/options";
import { SortableTable } from "../table/SortableTable";
import type { KillFeedEntry } from "./match";

const killColumnHelper = createMRTColumnHelper<KillFeedEntry>();
const defaultKillOptions = createDefaultTableOptions<KillFeedEntry>();

const KillerCell = ({ steamId, players }: { steamId: string; players: Record<string, PersonDisplay> }) => {
	if (emptySteamId(steamId)) {
		return <Typography variant="body2">World</Typography>;
	}
	const p = players[steamId];
	if (!p) {
		return <Typography variant="body2">{steamId}</Typography>;
	}
	return <PersonCell steamId={p.steamId} avatarHash={p.avatarHash} personaName={p.name} />;
};

const emptySteamId = (steamId: string): boolean => {
	return steamId === "" || steamId === "0";
};

export const KillFeedTable = ({
	kills,
	players,
}: {
	kills: KillFeedEntry[];
	players: Record<string, PersonDisplay>;
}) => {
	const columns = useMemo(
		() => [
			killColumnHelper.accessor("round", {
				...shortHeader("Round", "Round number"),
				size: 70,
				grow: false,
				Cell: ({ cell }) => {
					const round = cell.getValue<number>();
					return round > 0 ? round : "—";
				},
			}),
			killColumnHelper.accessor("tick", {
				...shortHeader("Tick", "Demo tick of the kill"),
				size: 80,
				grow: false,
			}),
			killColumnHelper.accessor("killerSteamId", {
				...shortHeader("Killer", "Killer"),
				grow: true,
				Cell: ({ cell }) => <KillerCell steamId={cell.getValue<string>()} players={players} />,
			}),
			killColumnHelper.accessor("weapon", {
				...shortHeader("Weapon", "Kill weapon"),
				grow: true,
			}),
			killColumnHelper.accessor("victimSteamId", {
				...shortHeader("Victim", "Victim"),
				grow: true,
				Cell: ({ cell }) => <KillerCell steamId={cell.getValue<string>()} players={players} />,
			}),
		],
		[players],
	);

	const table = useMaterialReactTable({
		...defaultKillOptions,
		columns,
		data: kills,
		enableColumnActions: false,
		initialState: {
			...defaultKillOptions.initialState,
			pagination: { pageIndex: 0, pageSize: 25 },
		},
	});

	if (kills.length === 0) {
		return null;
	}

	return <SortableTable table={table} title={"Kill Feed"} iconLeft={<GavelIcon />} />;
};
