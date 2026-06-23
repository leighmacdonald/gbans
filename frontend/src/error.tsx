import { ConnectError } from "@connectrpc/connect";
import type { AlertProps } from "@mui/material/Alert";
import Typography from "@mui/material/Typography";
import Box from "@mui/system/Box";
import { emptyOrNullString } from "./util/types.ts";

export type ApiError = {
	type: string;
	title: string;
	status: number;
	detail: string;
	instance: string;
	timestamp: string;
};

export enum ErrorCode {
	InvalidMimetype,
	DependencyMissing,
	PermissionDenied,
	Unknown,
	LoginRequired,
	NotFound = 5,
}

export class AppError extends Error {
	public code: ErrorCode;

	constructor(code: ErrorCode, message: string = "", options?: never) {
		if (emptyOrNullString(message)) {
			switch (code) {
				case ErrorCode.InvalidMimetype:
					message = "Forbidden file format (mimetype)";
					break;
				case ErrorCode.DependencyMissing:
					message = "Dependency missing, cannot continue";
					break;
				case ErrorCode.PermissionDenied:
					message = "Permission Denied";
					break;
				case ErrorCode.LoginRequired:
					message = "Please Login";
					break;
				case ErrorCode.NotFound:
					message = "Not Found";
					break;
				default:
					message = "🤯 🤯 🤯 Something went wrong 🤯 🤯 🤯";
			}
		}

		super(options);
		this.message = message;
		this.code = code;
	}
}

export function isApiError(err: unknown): err is ApiError {
	return (err as ApiError).instance !== undefined;
}

export const renderTableError = (err: unknown): AlertProps | undefined => {
	if (!err) {
		return undefined;
	}
	if (err instanceof ConnectError) {
		// TODO
		//const localized = err.findDetails(LocalizedMessageSchema).find((i) => i.locale === navigator.language);
		return {
			color: "error",
			children: (
				<Box>
					<Typography> {err.message}</Typography>
				</Box>
			),
		};
	} else {
		return { color: "error", children: String(err) };
	}
};
