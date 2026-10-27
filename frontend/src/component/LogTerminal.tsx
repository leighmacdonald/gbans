import { createClient } from "@connectrpc/connect";
import FilterListIcon from "@mui/icons-material/FilterList";
import PauseIcon from "@mui/icons-material/Pause";
import PlayArrowIcon from "@mui/icons-material/PlayArrow";
import SearchIcon from "@mui/icons-material/Search";
import VerticalAlignBottomIcon from "@mui/icons-material/VerticalAlignBottom";
import Box from "@mui/material/Box";
import Chip from "@mui/material/Chip";
import IconButton from "@mui/material/IconButton";
import InputAdornment from "@mui/material/InputAdornment";
import MenuItem from "@mui/material/MenuItem";
import Select from "@mui/material/Select";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { useVirtualizer } from "@tanstack/react-virtual";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ContainerWithHeader } from "../component/ContainerWithHeader.tsx";
import { type ServerInfoSafe, ServersService, type StreamLogsResponse } from "../rpc/servers/v1/servers_pb.ts";
import { finalTransport } from "../transport.ts";
import { categoryColour, categoryForLogType, LOG_CATEGORIES, type LogCategory } from "../util/logTypes.ts";
import { serverSelectOptions } from "../util/servers.ts";

const MAX_LINES = 10000;
const ROW_HEIGHT = 20;
const MONO_FONT = 'ui-monospace, "SF Mono", SFMono-Regular, Menlo, Consolas, "Liberation Mono", monospace';

interface LogTerminalProps {
	servers: ServerInfoSafe[];
	serverId: number;
	onServerChange: (serverId: number) => void;
}

export const LogTerminal = ({ servers, serverId, onServerChange }: LogTerminalProps) => {
	const client = useMemo(() => createClient(ServersService, finalTransport), []);

	const [lines, setLines] = useState<StreamLogsResponse[]>([]);
	const [paused, setPaused] = useState(false);
	const [dropped, setDropped] = useState(0);
	const [autoScroll, setAutoScroll] = useState(true);
	const [activeCategories, setActiveCategories] = useState<LogCategory[]>(() => LOG_CATEGORIES.map((c) => c.id));
	const [search, setSearch] = useState("");
	const [streamError, setStreamError] = useState<string | null>(null);

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

	useEffect(() => {
		const controller = new AbortController();
		linesRef.current = [];
		setLines([]);
		setDropped(0);
		setStreamError(null);
		setPaused(false);
		setAutoScroll(true);

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
										{line.body}
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
		</Box>
	);
};
