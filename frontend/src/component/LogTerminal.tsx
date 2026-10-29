import { createClient } from "@connectrpc/connect";
import { useMutation } from "@connectrpc/connect-query";
import FilterListIcon from "@mui/icons-material/FilterList";
import PauseIcon from "@mui/icons-material/Pause";
import PlayArrowIcon from "@mui/icons-material/PlayArrow";
import SearchIcon from "@mui/icons-material/Search";
import SendIcon from "@mui/icons-material/Send";
import TerminalIcon from "@mui/icons-material/Terminal";
import VerticalAlignBottomIcon from "@mui/icons-material/VerticalAlignBottom";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import IconButton from "@mui/material/IconButton";
import InputAdornment from "@mui/material/InputAdornment";
import Menu from "@mui/material/Menu";
import MenuItem from "@mui/material/MenuItem";
import Select from "@mui/material/Select";
import Stack from "@mui/material/Stack";
import type { SxProps, Theme } from "@mui/material/styles";
import TextField from "@mui/material/TextField";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { useVirtualizer } from "@tanstack/react-virtual";
import {
	type CSSProperties,
	type MouseEvent as ReactMouseEvent,
	useCallback,
	useEffect,
	useMemo,
	useRef,
	useState,
} from "react";
import { ContainerWithHeader } from "../component/ContainerWithHeader.tsx";
import { PlayerLinksMenu } from "../component/PlayerLinksMenu.tsx";
import { useAuth } from "../hooks/useAuth.ts";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import { Permission } from "../rpc/roles/v1/roles_pb.ts";
import { type ServerInfoSafe, ServersService, type StreamLogsResponse } from "../rpc/servers/v1/servers_pb.ts";
import { execCommand } from "../rpc/servers/v1/servers-ServersService_connectquery.ts";
import { finalTransport } from "../transport.ts";
import { type LogToken, type LogTokenKind, tokenizeLogLine } from "../util/logHighlight.ts";
import { categoryColour, categoryForLogType, LOG_CATEGORIES, type LogCategory } from "../util/logTypes.ts";
import { serverSelectOptions } from "../util/servers.ts";

const MAX_LINES = 10000;
const ROW_HEIGHT = 20;
const MONO_FONT = 'ui-monospace, "SF Mono", SFMono-Regular, Menlo, Consolas, "Liberation Mono", monospace';

// A player reference looks like "Hacksaw<12><[U:1:68745073]><Red>" inside quotes;
// the name is everything before the first tag.
const playerNameFromToken = (text: string) => text.slice(1, -1).split("<")[0];

// Per-token colors layered on top of the line's category color, so names,
// quoted values and numbers stand out without losing the type signal.
const TOKEN_STYLES: Partial<Record<LogTokenKind, CSSProperties>> = {
	timestamp: { color: "#6e7681" },
	player: { color: "#7ee787", fontWeight: 600 },
	quoted: { color: "#79c0ff" },
	steamid: { color: "#6e7681" },
	address: { color: "#6e7681" },
	number: { color: "#ffa657" },
	keyword: { fontWeight: 600 },
};

// Player references carrying a resolvable SteamID are clickable and open the
// player's quick-link menu.
const playerTokenStyle: SxProps<Theme> = {
	...TOKEN_STYLES.player,
	display: "inline",
	cursor: "pointer",
	"&:hover": {
		textDecoration: "underline",
	},
};

interface LogTerminalProps {
	servers: ServerInfoSafe[];
	serverId: number;
	onServerChange: (serverId: number) => void;
}

export const LogTerminal = ({ servers, serverId, onServerChange }: LogTerminalProps) => {
	const client = useMemo(() => createClient(ServersService, finalTransport), []);
	const { hasPermission } = useAuth();
	const { sendError } = useUserFlashCtx();

	const [lines, setLines] = useState<StreamLogsResponse[]>([]);
	const [paused, setPaused] = useState(false);
	const [dropped, setDropped] = useState(0);
	const [autoScroll, setAutoScroll] = useState(true);
	const [activeCategories, setActiveCategories] = useState<LogCategory[]>(() => LOG_CATEGORIES.map((c) => c.id));
	const [search, setSearch] = useState("");
	const [streamError, setStreamError] = useState<string | null>(null);
	const [playerMenu, setPlayerMenu] = useState<{ anchorEl: HTMLElement; steamId: string; name: string } | null>(null);
	const [command, setCommand] = useState("");
	const [commandResponse, setCommandResponse] = useState<string | null>(null);

	const linesRef = useRef<StreamLogsResponse[]>([]);
	const pausedRef = useRef(paused);
	const autoScrollRef = useRef(autoScroll);
	const renderPending = useRef(false);
	const scrollRef = useRef<HTMLDivElement>(null);

	useEffect(() => {
		pausedRef.current = paused;
	}, [paused]);

	useEffect(() => {
		autoScrollRef.current = autoScroll;
	}, [autoScroll]);

	const scrollToBottom = useCallback(() => {
		const el = scrollRef.current;
		if (el) {
			el.scrollTop = el.scrollHeight;
		}
	}, []);

	const closePlayerMenu = useCallback(() => {
		setPlayerMenu(null);
	}, []);

	const openPlayerMenu = (event: ReactMouseEvent<HTMLElement>, steamId: string, name: string) => {
		event.stopPropagation();
		setPlayerMenu({ anchorEl: event.currentTarget, steamId, name });
	};

	// The raw [U:1:...] id is what the log lines actually contain, so it is the
	// most reliable substring to filter this server's buffered lines by.
	const filterToPlayer = (rawSteamId: string) => {
		setSearch(rawSteamId);
		closePlayerMenu();
	};

	const execCommandMutation = useMutation(execCommand, {
		onSuccess: (response) => {
			setCommandResponse(response.response);
			setCommand("");
		},
		onError: sendError,
	});

	const submitCommand = () => {
		const trimmed = command.trim();
		if (trimmed === "" || execCommandMutation.isPending) {
			return;
		}

		execCommandMutation.mutate({ serverId, command: trimmed });
	};

	const renderToken = (token: LogToken, index: number) => {
		if (token.kind === "player" && token.steamId) {
			const steamId = token.steamId;
			const openMenu = (event: ReactMouseEvent<HTMLElement>) =>
				openPlayerMenu(event, steamId, playerNameFromToken(token.text));
			return (
				<Box key={index} component="span" sx={playerTokenStyle} onClick={openMenu}>
					{token.text}
				</Box>
			);
		}
		return (
			<span key={index} style={TOKEN_STYLES[token.kind]}>
				{token.text}
			</span>
		);
	};

	useEffect(() => {
		const controller = new AbortController();
		linesRef.current = [];
		setLines([]);
		setDropped(0);
		setStreamError(null);
		setPaused(false);
		setAutoScroll(true);
		setPlayerMenu(null);
		setCommandResponse(null);

		const scheduleRender = () => {
			// Batch appends into at most one re-render per animation frame.
			if (renderPending.current) {
				return;
			}
			renderPending.current = true;
			requestAnimationFrame(() => {
				renderPending.current = false;
				setLines(linesRef.current.slice());
				if (autoScrollRef.current) {
					// A second frame ensures the scroll happens after React has
					// committed the new rows.
					requestAnimationFrame(scrollToBottom);
				}
			});
		};

		void (async () => {
			try {
				const stream = await client.streamLogs({ serverId }, { signal: controller.signal });
				for await (const line of stream) {
					if (pausedRef.current) {
						setDropped((d) => d + 1);
						continue;
					}
					linesRef.current.push(line);
					if (linesRef.current.length > MAX_LINES) {
						linesRef.current.splice(0, linesRef.current.length - MAX_LINES);
					}
					scheduleRender();
				}
			} catch (error) {
				if (!controller.signal.aborted) {
					setStreamError(error instanceof Error ? error.message : "stream failed");
				}
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
		const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 24;
		if (!atBottom && autoScroll) {
			setAutoScroll(false);
		}
	};

	const visibleLines = useMemo(() => {
		const query = search.trim().toLowerCase();
		return lines.filter((line) => {
			if (!activeCategories.includes(categoryForLogType(line.eventType))) {
				return false;
			}
			if (query && !line.body.toLowerCase().includes(query)) {
				return false;
			}
			return true;
		});
	}, [lines, activeCategories, search]);

	const virtualizer = useVirtualizer({
		count: visibleLines.length,
		getScrollElement: () => scrollRef.current,
		estimateSize: () => ROW_HEIGHT,
		fixedSize: true,
		overscan: 15,
	});

	const togglePaused = () => {
		setDropped(0);
		setPaused((p) => !p);
	};

	const toggleAutoScroll = () => {
		const next = !autoScroll;
		setAutoScroll(next);
		if (next) {
			scrollToBottom();
		}
	};

	const toggleCategory = (id: LogCategory) => {
		setActiveCategories((prev) => (prev.includes(id) ? prev.filter((c) => c !== id) : [...prev, id]));
	};

	return (
		<Box>
			<ContainerWithHeader title={"Filters"} iconLeft={<FilterListIcon />} padding={2}>
				<Stack
					direction={{ xs: "column", md: "row" }}
					spacing={2}
					alignItems={{ xs: "stretch", md: "center" }}
					justifyContent="space-between"
				>
					<Stack direction="row" spacing={2}>
						<Select
							label="Server"
							value={serverId}
							onChange={(e) => onServerChange(Number(e.target.value))}
							size="small"
							sx={{ minWidth: 200 }}
						>
							{serverSelectOptions(servers).map((server) => (
								<MenuItem key={server.value} value={server.value}>
									{server.label}
								</MenuItem>
							))}
						</Select>
						<TextField
							label="Filter lines"
							size="small"
							placeholder="Search…"
							value={search}
							onChange={(e) => setSearch(e.target.value)}
							sx={{ minWidth: 200, flex: 1, maxWidth: 360 }}
							InputProps={{
								startAdornment: (
									<InputAdornment position="start">
										<SearchIcon fontSize="small" />
									</InputAdornment>
								),
							}}
						/>
					</Stack>
					<Stack direction="row" spacing={1} alignItems="center" flexWrap="wrap">
						{LOG_CATEGORIES.map((category) => {
							const active = activeCategories.includes(category.id);
							return (
								<Tooltip
									key={category.id}
									title={active ? `Hide ${category.label}` : `Show ${category.label}`}
								>
									<Chip
										label={category.label}
										size="small"
										onClick={() => toggleCategory(category.id)}
										sx={{
											border: `1px solid ${category.colour}`,
											color: active ? "#0d1117" : category.colour,
											backgroundColor: active ? category.colour : "transparent",
											opacity: active ? 1 : 0.55,
										}}
									/>
								</Tooltip>
							);
						})}
						<Tooltip title={paused ? "Resume updates" : "Pause updates"}>
							<IconButton onClick={togglePaused} color={paused ? "warning" : "default"}>
								{paused ? <PlayArrowIcon /> : <PauseIcon />}
							</IconButton>
						</Tooltip>
						<Tooltip title={autoScroll ? "Auto-scroll on" : "Auto-scroll off"}>
							<IconButton onClick={toggleAutoScroll} color={autoScroll ? "primary" : "default"}>
								<VerticalAlignBottomIcon />
							</IconButton>
						</Tooltip>
					</Stack>
				</Stack>
			</ContainerWithHeader>

			<Box
				ref={scrollRef}
				onScroll={handleScroll}
				sx={{
					overflow: "auto",
					backgroundColor: "#0d1117",
					borderRadius: 1,
					height: "calc(100vh - 400px)",
					minHeight: 400,
					mt: 2,
					fontFamily: MONO_FONT,
				}}
			>
				{visibleLines.length === 0 ? (
					<Box sx={{ p: 2 }}>
						<Typography variant="body2" sx={{ color: "#5c6773" }}>
							Waiting for log output…
						</Typography>
					</Box>
				) : (
					<Box
						sx={{
							position: "relative",
							height: virtualizer.getTotalSize(),
							width: "max-content",
							minWidth: "100%",
						}}
					>
						{virtualizer.getVirtualItems().map((item) => {
							const line = visibleLines[item.index];
							return (
								<Box
									key={item.key}
									sx={{
										position: "absolute",
										top: 0,
										left: 0,
										transform: `translateY(${item.start}px)`,
										whiteSpace: "pre",
										px: 1,
									}}
								>
									<Typography
										component="span"
										sx={{
											color: categoryColour(categoryForLogType(line.eventType)),
											fontSize: 13,
											fontFamily: "inherit",
											lineHeight: `${ROW_HEIGHT}px`,
											display: "block",
											whiteSpace: "pre",
										}}
									>
										{tokenizeLogLine(line.body).map(renderToken)}
									</Typography>
								</Box>
							);
						})}
					</Box>
				)}
			</Box>

			<Stack direction="row" spacing={2} alignItems="center" sx={{ mt: 1 }}>
				<Typography variant="caption">{lines.length.toLocaleString()} lines buffered</Typography>
				{paused && (
					<Typography variant="caption" color="warning.main">
						Paused{dropped > 0 ? ` — ${dropped.toLocaleString()} new lines skipped` : ""}
					</Typography>
				)}
				{streamError && (
					<Typography variant="caption" color="error.main">
						Stream error: {streamError}
					</Typography>
				)}
			</Stack>

			{hasPermission(Permission.SERVER_WRITE) && (
				<ContainerWithHeader title={"RCON"} iconLeft={<TerminalIcon />} padding={2} spacing={2}>
					<Stack direction="row" spacing={1}>
						<TextField
							value={command}
							onChange={(event) => setCommand(event.target.value)}
							onKeyDown={(event) => {
								if (event.key === "Enter" && !execCommandMutation.isPending && command.trim() !== "") {
									event.preventDefault();
									submitCommand();
								}
							}}
							placeholder="rcon command (e.g. sm_listme)"
							size="small"
							fullWidth
							disabled={execCommandMutation.isPending}
						/>
						<Button
							variant="contained"
							startIcon={<SendIcon />}
							onClick={submitCommand}
							disabled={command.trim() === "" || execCommandMutation.isPending}
						>
							Send
						</Button>
					</Stack>
					{commandResponse !== null && (
						<Typography
							variant="body2"
							sx={{
								fontFamily: MONO_FONT,
								whiteSpace: "pre-wrap",
								wordBreak: "break-word",
							}}
						>
							{commandResponse === "" ? "(no response)" : commandResponse}
						</Typography>
					)}
				</ContainerWithHeader>
			)}

			<Menu
				anchorEl={playerMenu?.anchorEl ?? null}
				id="log-player-menu"
				open={playerMenu !== null}
				onClose={closePlayerMenu}
				onClick={closePlayerMenu}
				transformOrigin={{ horizontal: "left", vertical: "top" }}
				anchorOrigin={{ horizontal: "left", vertical: "bottom" }}
			>
				{playerMenu ? (
					<PlayerLinksMenu
						steamId={playerMenu.steamId}
						personaName={playerMenu.name}
						onAction={closePlayerMenu}
						onFilterPlayer={filterToPlayer}
					/>
				) : null}
			</Menu>
		</Box>
	);
};
