import Typography from "@mui/material/Typography";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useMemo } from "react";
import type { PersonDisplay } from "../../rpc/person/v1/person_core_pb";
import { formatMatchClock } from "../../util/time.ts";
import { PersonCell } from "../PersonCell";
import { createDefaultTableOptions } from "../table/options";
import { SortableTable } from "../table/SortableTable";
import type { ChatFeedEntry } from "./match";

const chatColumnHelper = createMRTColumnHelper<ChatFeedEntry>();
const defaultChatOptions = createDefaultTableOptions<ChatFeedEntry>();

export const MatchChatTable = ({
	chat,
	players,
}: {
	chat: ChatFeedEntry[];
	players: Record<string, PersonDisplay>;
}) => {
	const columns = useMemo(
		() => [
			chatColumnHelper.accessor("round", {
				header: "Round",
				size: 70,
				grow: false,
				Cell: ({ cell }) => {
					const round = cell.getValue<number>();
					return round > 0 ? round : "—";
				},
			}),
			chatColumnHelper.accessor("tick", {
				header: "Tick",
				size: 80,
				grow: false,
			}),
			chatColumnHelper.accessor("ticksSinceStart", {
				header: "Time",
				size: 80,
				grow: false,
				Cell: ({ cell }) => formatMatchClock(cell.getValue<number>()),
			}),
			chatColumnHelper.accessor("steamId", {
				header: "Player",
				grow: false,
				Cell: ({ row }) => {
					const p: PersonDisplay | undefined = players[row.original.steamId];
					if (!p) {
						return <Typography variant="body2">{row.original.name}</Typography>;
					}
					return <PersonCell steamId={p.steamId} avatarHash={p.avatarHash} personaName={p.name} />;
				},
			}),
			chatColumnHelper.accessor("body", {
				header: "Message",
				grow: true,
			}),
		],
		[players],
	);

	const table = useMaterialReactTable({
		...defaultChatOptions,
		columns,
		data: chat,
		enableColumnActions: false,
		initialState: {
			...defaultChatOptions.initialState,
			pagination: { pageIndex: 0, pageSize: 25 },
		},
	});

	if (chat.length === 0) {
		return null;
	}

	return <SortableTable table={table} title={"Match Chat"} />;
};
