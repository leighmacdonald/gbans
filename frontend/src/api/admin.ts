import type { Config } from "../schema/config.ts";
import { apiCall } from "./common.ts";

export const apiSaveSettings = async (signal: AbortSignal, settings: Config) => {
	return await apiCall(signal, `/api/config`, "PUT", settings);
};

export const apiGetSettings = async (signal: AbortSignal) => {
	return await apiCall<Config>(signal, "/api/config", "GET");
};
