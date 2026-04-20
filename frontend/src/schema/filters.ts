import {FilterAction} from "../rpc/chat/v1/wordfilter_pb.ts";

export const FilterActionCollection = [FilterAction.KICK_UNSPECIFIED, FilterAction.MUTE, FilterAction.BAN];

export const filterActionString = (fa: FilterAction) => {
	switch (fa) {
		case FilterAction.BAN:
			return "Ban";
		case FilterAction.KICK_UNSPECIFIED:
			return "Kick";
		case FilterAction.MUTE:
			return "Mute";
	}
};
