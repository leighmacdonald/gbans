import { describe, expect, test } from "vitest";
import type { ServerInfoSafe } from "../rpc/servers/v1/servers_pb.ts";
import { serverSelectOptions, sortServers } from "./servers.ts";

const server = (serverId: number, serverName: string): ServerInfoSafe =>
	({ serverId, serverName }) as unknown as ServerInfoSafe;

describe("sortServers", () => {
	test("sorts by server name", () => {
		expect(sortServers([server(3, "zulu"), server(1, "alpha"), server(2, "mike")])).toEqual([
			server(1, "alpha"),
			server(2, "mike"),
			server(3, "zulu"),
		]);
	});

	test("handles undefined", () => {
		expect(sortServers(undefined)).toEqual([]);
	});
});

describe("serverSelectOptions", () => {
	test("returns sorted options", () => {
		expect(serverSelectOptions([server(2, "zulu"), server(1, "alpha")])).toEqual([
			{ label: "alpha", value: 1 },
			{ label: "zulu", value: 2 },
		]);
	});

	test("handles undefined", () => {
		expect(serverSelectOptions(undefined)).toEqual([]);
	});
});
