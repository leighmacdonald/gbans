import { Code, ConnectError } from "@connectrpc/connect";
import type { AlertProps } from "@mui/material/Alert";
import Typography from "@mui/material/Typography";
import Box from "@mui/system/Box";
import { type Violation, ViolationsSchema } from "./rpc/buf/validate/validate_pb.ts";
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

const maxViolationsShown = 3;

const humanizeFieldName = (name: string): string =>
	name
		.split("_")
		.filter((w) => w !== "")
		.map((w) => (w.toLowerCase() === "id" ? "ID" : w.charAt(0).toUpperCase() + w.slice(1).toLowerCase()))
		.join(" ");

const describeViolation = (violation: Violation): string => {
	const path = (violation.field?.elements ?? [])
		.map((element) => humanizeFieldName(element.fieldName))
		.filter((part) => part !== "")
		.join(" > ");
	return path ? `${path}: ${violation.message}` : violation.message;
};

// friendlyError converts raw thrown errors into a heading/message pair suitable
// for display to end users. In particular, protobuf validation failures carry
// machine-oriented field paths ("target_id: value is required [required]")
// which are rewritten as plain per-field messages.
export const friendlyError = (error: unknown): { heading: string; message: string } => {
	if (isApiError(error)) {
		return { heading: error.title, message: error.detail };
	}
	if (error instanceof ConnectError) {
		if (error.code === Code.InvalidArgument) {
			const violations = error.findDetails(ViolationsSchema).flatMap((detail) => detail.violations);
			if (violations.length > 0) {
				const shown = violations.slice(0, maxViolationsShown).map(describeViolation).join("; ");
				const remaining = violations.length - maxViolationsShown;
				return {
					heading: "Invalid input",
					message: remaining > 0 ? `${shown}; and ${remaining} more` : shown,
				};
			}
			return { heading: "Invalid input", message: error.rawMessage };
		}
		switch (error.code) {
			case Code.Unauthenticated:
				return { heading: "Login required", message: error.rawMessage || "Please log in and try again." };
			case Code.PermissionDenied:
				return { heading: "Permission denied", message: error.rawMessage };
			case Code.NotFound:
				return { heading: "Not found", message: error.rawMessage };
			case Code.AlreadyExists:
				return { heading: "Already exists", message: error.rawMessage };
			case Code.FailedPrecondition:
				return { heading: "Action not allowed", message: error.rawMessage };
			case Code.ResourceExhausted:
				return { heading: "Too many requests", message: error.rawMessage };
			case Code.Unavailable:
				return { heading: "Service unavailable", message: error.rawMessage };
			case Code.DeadlineExceeded:
				return { heading: "Request timed out", message: error.rawMessage };
			default:
				return { heading: "Request failed", message: error.rawMessage };
		}
	}
	if (error instanceof Error) {
		return { heading: error.name, message: error.message };
	}
	return { heading: "Error", message: String(error) };
};

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
