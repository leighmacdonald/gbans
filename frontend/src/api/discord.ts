import type { CallbackLink } from "../schema/query.ts";
import { apiCall } from "./common.ts";

export const apiGetDiscordLogin = async (signal: AbortSignal) => {
	return await apiCall<CallbackLink>(signal, "/api/discord/login");
};
