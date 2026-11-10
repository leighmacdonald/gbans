import { FilterAction } from "../rpc/chat/v1/wordfilter_pb.ts";

// FilterAction's zero value is named KICK_UNSPECIFIED by proto convention, but it is
// the actual kick action, so map it to a presentable label for UI display.
export const filterActionLabel = (action: number): string => {
	switch (action) {
		case FilterAction.KICK_UNSPECIFIED:
			return "Kick";
		case FilterAction.MUTE:
			return "Mute";
		case FilterAction.BAN:
			return "Ban";
		default:
			return FilterAction[action] ?? String(action);
	}
};
