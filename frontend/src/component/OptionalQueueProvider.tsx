import type { PropsWithChildren } from "react";
import { useAuth } from "../hooks/useAuth";
import { QueueProvider } from "./queue/QueueProvider";

export const OptionalQueueProvider = ({ children }: PropsWithChildren) => {
	const { isAuthenticated } = useAuth();
	if (isAuthenticated()) {
		return <QueueProvider>{children}</QueueProvider>;
	}
	return children;
};
