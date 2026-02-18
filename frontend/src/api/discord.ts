import type { CallbackLink } from "../schema/query.ts";
import { apiCall } from "./common.ts";

export const apiGetDiscordLogin = async () => {
	return apiCall<CallbackLink>("/api/discord/login");
};
