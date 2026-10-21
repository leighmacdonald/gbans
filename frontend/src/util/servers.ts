import type { ServerInfoSafe } from "../rpc/servers/v1/servers_pb.ts";

export const sortServers = (servers?: ServerInfoSafe[]): ServerInfoSafe[] =>
	(servers ?? []).toSorted((a, b) => a.serverName.localeCompare(b.serverName));

export const serverSelectOptions = (servers?: ServerInfoSafe[]): { label: string; value: number }[] =>
	sortServers(servers).map((server) => ({ label: server.serverName, value: server.serverId }));
