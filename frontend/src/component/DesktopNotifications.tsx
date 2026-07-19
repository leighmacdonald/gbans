import { useEffect, useState } from "react";
import * as engineer from "../icons/engineer_blu.jpg";
import type { UserNotification } from "../rpc/notification/v1/notification_pb.ts";

export const DesktopNotifications = ({
	notifications,
	isLoading,
}: {
	notifications?: UserNotification[];
	isLoading: boolean;
}) => {
	const [newest, setNewest] = useState<string>();

	useEffect(() => {
		if (isLoading || notifications == null) {
			return;
		}

		// Track the newest one we get on initial load so we are only showing items that are newer.
		if (newest == null) {
			setNewest(notifications.length > 0 ? notifications[0].personNotificationId : "");
			return;
		}

		const opened = notifications
			.filter((n) => n.personNotificationId > newest)
			.map((n) => {
				setNewest(n.personNotificationId);
				return new Notification("New Notification Received", {
					body: n.message,
					// timestamp: Math.floor(n.created_on.getTime()), chrome only
					silent: true,
					lang: "en-US",
					icon: engineer.default,
				});
			});
		return () => {
			opened.map((n) => n.close());
		};
	}, [isLoading, newest, notifications]);

	return null;
};
