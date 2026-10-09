import EventIcon from "@mui/icons-material/Event";
import Typography from "@mui/material/Typography";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useMemo } from "react";
import type { PersonDisplay } from "../../rpc/person/v1/person_core_pb";
import { formatMatchClock } from "../../util/time.ts";
import { PersonCell } from "../PersonCell";
import { shortHeader } from "../table/columnHeaders";
import { createDefaultTableOptions } from "../table/options";
import { SortableTable } from "../table/SortableTable";
import type { EventFeedEntry } from "./match";

const eventColumnHelper = createMRTColumnHelper<EventFeedEntry>();
const defaultEventOptions = createDefaultTableOptions<EventFeedEntry>();

export const humanizeEventType = (eventType: string): string =>
	eventType
		.split("_")
		.map((part) => (part.length > 0 ? part[0].toUpperCase() + part.slice(1) : part))
		.join(" ");

const ActorCell = ({ steamId, players }: { steamId: string; players: Record<string, PersonDisplay> }) => {
	if (steamId === "" || steamId === "0") {
		return <Typography variant="body2">—</Typography>;
	}
	const p = players[steamId];
	if (!p) {
		return <Typography variant="body2">{steamId}</Typography>;
	}
	return <PersonCell steamId={p.steamId} avatarHash={p.avatarHash} personaName={p.name} />;
};

export const EventFeedTable = ({
	events,
	players,
}: {
	events: EventFeedEntry[];
	players: Record<string, PersonDisplay>;
}) => {
	const columns = useMemo(
		() => [
			eventColumnHelper.accessor("eventType", {
				...shortHeader("Type", "Match event type"),
				size: 150,
				grow: false,
				filterVariant: "select",
				Cell: ({ cell }) => humanizeEventType(cell.getValue<string>()),
			}),
			eventColumnHelper.accessor("round", {
				...shortHeader("Round", "Round number"),
				size: 70,
				grow: false,
				Cell: ({ cell }) => {
					const round = cell.getValue<number>();
					return round > 0 ? round : "—";
				},
			}),
			eventColumnHelper.accessor("ticksSinceStart", {
				...shortHeader("Time", "Match time of the event"),
				size: 80,
				grow: false,
				Cell: ({ cell }) => formatMatchClock(cell.getValue<number>()),
			}),
			eventColumnHelper.accessor("actorSteamId", {
				...shortHeader("Actor", "Primary actor"),
				grow: true,
				Cell: ({ cell }) => <ActorCell steamId={cell.getValue<string>()} players={players} />,
			}),
			eventColumnHelper.accessor("targetSteamId", {
				...shortHeader("Target", "Primary target"),
				grow: true,
				Cell: ({ cell }) => <ActorCell steamId={cell.getValue<string>()} players={players} />,
			}),
			eventColumnHelper.accessor("summary", {
				...shortHeader("Details", "Event details"),
				grow: true,
			}),
		],
		[players],
	);

	const table = useMaterialReactTable({
		...defaultEventOptions,
		columns,
		data: events,
		enableColumnActions: false,
		initialState: {
			...defaultEventOptions.initialState,
			pagination: { pageIndex: 0, pageSize: 10 },
		},
	});

	if (events.length === 0) {
		return null;
	}

	return <SortableTable table={table} title={"Event Feed"} iconLeft={<EventIcon />} />;
};
