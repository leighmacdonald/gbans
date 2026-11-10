import { timestampDate } from "@bufbuild/protobuf/wkt";
import { createClient } from "@connectrpc/connect";
import AllInclusiveIcon from "@mui/icons-material/AllInclusive";
import PauseIcon from "@mui/icons-material/Pause";
import PlayArrowIcon from "@mui/icons-material/PlayArrow";
import type { SelectChangeEvent } from "@mui/material";
import Box from "@mui/material/Box";
import Chip from "@mui/material/Chip";
import FormControl from "@mui/material/FormControl";
import IconButton from "@mui/material/IconButton";
import InputAdornment from "@mui/material/InputAdornment";
import InputLabel from "@mui/material/InputLabel";
import MenuItem from "@mui/material/MenuItem";
import Select from "@mui/material/Select";
import { type SxProps, type Theme, useTheme } from "@mui/material/styles";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { format } from "date-fns/format";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ChatService, type StreamChatResponse } from "../rpc/chat/v1/chat_pb.ts";
import type { ServerInfoSafe } from "../rpc/servers/v1/servers_pb.ts";
import { finalTransport } from "../transport.ts";
import { serverSelectOptions } from "../util/servers.ts";
import { PersonCell } from "./PersonCell.tsx";

const MAX_MESSAGES = 300;
const HISTORY_LIMIT = 50;

/** Formats a streamed timestamp as a clock time, which is what a live feed needs. */
const renderClockTime = (createdOn?: Date) => (createdOn ? format(createdOn, "HH:mm:ss") : "");

interface ChatFeedProps {
	servers: ServerInfoSafe[];
}

/**
 * Live in-game chat feed. Streams messages from all servers, or a selected
 * one, replaying a short buffer of recent history before continuing live.
 */
export const ChatFeed = ({ servers }: ChatFeedProps) => {
	const client = useMemo(() => createClient(ChatService, finalTransport), []);
	const theme = useTheme();

	const [serverId, setServerId] = useState(0); // 0 = all servers
	const [messages, setMessages] = useState<StreamChatResponse[]>([]);
	const [paused, setPaused] = useState(false);
	const [dropped, setDropped] = useState(0);
	const [streamError, setStreamError] = useState<string | null>(null);

	const messagesRef = useRef<StreamChatResponse[]>([]);
	const lastSeqRef = useRef<bigint>(0n);
	const pausedRef = useRef(paused);
	// Follow new messages only while the view is pinned to the bottom.
	const autoScrollRef = useRef(true);
	const renderPending = useRef(false);
	const scrollRef = useRef<HTMLDivElement>(null);

	useEffect(() => {
		pausedRef.current = paused;
	}, [paused]);

	const scrollToBottom = useCallback(() => {
		const el = scrollRef.current;
		if (el) {
			el.scrollTop = el.scrollHeight;
		}
	}, []);

	useEffect(() => {
		const controller = new AbortController();
		messagesRef.current = [];
		lastSeqRef.current = 0n;
		setMessages([]);
		setDropped(0);
		setStreamError(null);
		setPaused(false);
		autoScrollRef.current = true;

		const scheduleRender = () => {
			// Batch appends into at most one re-render per animation frame.
			if (renderPending.current) {
				return;
			}
			renderPending.current = true;
			requestAnimationFrame(() => {
				renderPending.current = false;
				setMessages(messagesRef.current.slice());
				if (autoScrollRef.current) {
					requestAnimationFrame(scrollToBottom);
				}
			});
		};

		const sleep = (ms: number) =>
			new Promise<void>((resolve) => {
				const timer = setTimeout(() => {
					controller.signal.removeEventListener("abort", onAbort);
					resolve();
				}, ms);
				const onAbort = () => {
					clearTimeout(timer);
					resolve();
				};
				controller.signal.addEventListener("abort", onAbort, { once: true });
			});

		void (async () => {
			let attempt = 0;
			// The stream is expected to be cut by proxies and idle timeouts; always
			// re-establish it, replaying history deduped against the last seen seq.
			while (!controller.signal.aborted) {
				try {
					const stream = await client.streamChat(
						{
							serverIds: serverId > 0 ? [serverId] : [],
							historyLimit: HISTORY_LIMIT,
						},
						{ signal: controller.signal },
					);
					setStreamError(null);
					attempt = 0;
					let firstOfStream = true;
					for await (const msg of stream) {
						// Keepalives only exist to stop intermediaries from dropping
						// the connection while chat is quiet.
						if (msg.keepalive) {
							continue;
						}
						if (firstOfStream) {
							firstOfStream = false;
							// A restarted backend restarts seq numbering, so replayed
							// history would otherwise be discarded as "already seen".
							if (msg.seq < lastSeqRef.current) {
								lastSeqRef.current = 0n;
							}
						}
						// History is replayed on reconnect; skip anything already shown.
						if (msg.seq <= lastSeqRef.current) {
							continue;
						}
						lastSeqRef.current = msg.seq;
						if (pausedRef.current) {
							setDropped((d) => d + 1);
							continue;
						}
						messagesRef.current.push(msg);
						if (messagesRef.current.length > MAX_MESSAGES) {
							messagesRef.current.splice(0, messagesRef.current.length - MAX_MESSAGES);
						}
						scheduleRender();
					}
					if (controller.signal.aborted) {
						return;
					}
					setStreamError("Feed disconnected, reconnecting…");
				} catch (error) {
					if (controller.signal.aborted) {
						return;
					}
					const detail = error instanceof Error ? error.message : "stream failed";
					setStreamError(`Feed connection lost (${detail}), reconnecting…`);
				}
				const delay = Math.min(1000 * 2 ** attempt, 15_000) + Math.random() * 500;
				attempt += 1;
				await sleep(delay);
			}
		})();

		return () => {
			controller.abort();
		};
	}, [client, serverId, scrollToBottom]);

	const handleScroll = () => {
		const el = scrollRef.current;
		if (!el) {
			return;
		}
		// Re-enable following as soon as the user scrolls back to the bottom.
		autoScrollRef.current = el.scrollHeight - el.scrollTop - el.clientHeight < 24;
	};

	const togglePaused = () => {
		setDropped(0);
		setPaused((p) => !p);
	};

	const onServerChange = (event: SelectChangeEvent<number>) => {
		setServerId(Number(event.target.value));
	};

	const rowStyles: SxProps<Theme> = {
		display: "flex",
		alignItems: "center",
		gap: 1,
		px: 1.5,
		py: 0.75,
		borderBottom: `1px solid ${theme.palette.divider}`,
		"&:hover": { bgcolor: theme.palette.action.hover },
	};
	const bodyStyle: SxProps<Theme> = {
		flex: 1,
		wordBreak: "break-word",
		whiteSpace: "pre-wrap",
	};
	const metaStyle: SxProps<Theme> = {
		color: theme.palette.text.secondary,
		fontSize: "0.75rem",
		whiteSpace: "nowrap",
	};

	return (
		<Box sx={{ display: "flex", flexDirection: "column", gap: 1 }}>
			<Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
				<FormControl size="small" sx={{ minWidth: 220 }}>
					<InputLabel id="chat-feed-server-label">Server</InputLabel>
					<Select<number>
						labelId="chat-feed-server-label"
						value={serverId}
						label="Server"
						onChange={onServerChange}
						startAdornment={
							serverId === 0 ? (
								<InputAdornment position="start">
									<AllInclusiveIcon fontSize="small" />
								</InputAdornment>
							) : undefined
						}
					>
						<MenuItem value={0}>
							<Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
								<AllInclusiveIcon fontSize="inherit" />
								All servers
							</Box>
						</MenuItem>
						{serverSelectOptions(servers).map((opt) => (
							<MenuItem key={opt.value} value={opt.value}>
								{opt.label}
							</MenuItem>
						))}
					</Select>
				</FormControl>
				<Tooltip title={paused ? "Resume feed" : "Pause feed"}>
					<IconButton onClick={togglePaused} size="small">
						{paused ? <PlayArrowIcon /> : <PauseIcon />}
					</IconButton>
				</Tooltip>
				<Typography variant="caption" sx={{ color: theme.palette.text.secondary }}>
					{paused && dropped > 0 ? `${dropped} messages hidden` : `${messages.length} messages`}
				</Typography>
			</Box>
			{streamError ? (
				<Typography variant="caption" sx={{ color: theme.palette.warning.main }}>
					{streamError}
				</Typography>
			) : null}
			<Box
				ref={scrollRef}
				onScroll={handleScroll}
				sx={{
					border: `1px solid ${theme.palette.divider}`,
					borderRadius: 1,
					maxHeight: 400,
					overflowY: "auto",
				}}
			>
				{messages.length === 0 ? (
					<Typography variant="body2" color="textSecondary" sx={{ p: 2 }}>
						No chat captured yet. Messages appear here as servers send their logs.
					</Typography>
				) : (
					messages.map((msg) => (
						<Box key={msg.seq} sx={rowStyles}>
							<PersonCell
								steamId={msg.steamId}
								personaName={msg.personaName}
								avatarHash={msg.avatarHash}
							/>
							<Typography variant="body2" sx={bodyStyle}>
								{msg.body}
							</Typography>
							{msg.isTeam ? <Chip label="Team" size="small" color="warning" variant="outlined" /> : null}
							<Typography variant="caption" sx={{ ...metaStyle, width: 200 }} noWrap>
								{msg.serverName}
							</Typography>
							<Typography variant="caption" sx={metaStyle}>
								{renderClockTime(msg.createdOn ? timestampDate(msg.createdOn) : undefined)}
							</Typography>
						</Box>
					))
				)}
			</Box>
		</Box>
	);
};
