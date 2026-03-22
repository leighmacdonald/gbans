import type { DemoFile } from "../schema/demo.ts";
import { transformCreatedOnDate } from "../util/time.ts";
import { apiCall } from "./common";

export const apiGetDemos = async (signal: AbortSignal) => {
	const resp = await apiCall<DemoFile[]>(signal, "/api/demos", "POST");
	return resp.map(transformCreatedOnDate);
};

export const apiGetDemoCleanup = async (signal: AbortSignal) => await apiCall(signal, "/api/demos/cleanup");
