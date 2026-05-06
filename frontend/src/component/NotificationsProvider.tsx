import { type JSX, useState } from "react";
import { NotificationsCtx } from "../contexts/NotificationsCtx.tsx";
import type { UserNotification } from "../rpc/notification/v1/notification_pb.ts";

export const NotificationsProvider = ({ children }: { children: JSX.Element }) => {
	const [selectedIds, setSelectedIds] = useState<number[]>([]);
	const [notifications, setNotifications] = useState<UserNotification[]>([]);

	return (
		<NotificationsCtx.Provider
			value={{
				setNotifications,
				notifications,
				selectedIds,
				setSelectedIds,
			}}
		>
			{children}
		</NotificationsCtx.Provider>
	);
};
