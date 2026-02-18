import * as Sentry from "@sentry/react";
import { z } from "zod/v4";

export const runModeNames = z.enum(["development", "production"]);
export type runModeNames = z.infer<typeof runModeNames>;

export const runMode: runModeNames = (process.env.NODE_ENV as runModeNames) || "development";

export enum Level {
	info = 0,
	warn = 1,
	err = 2,
}

export const log = (msg: unknown, level: Level = Level.err): void => {
	Sentry.captureException(msg);

	if (runMode === "development") {
		if (Object.hasOwn(msg as object, "message") && (msg as Error).name !== "AbortError") {
			// eslint-disable-next-line no-console
			console.log(`[${level}] ${msg}`);
		}
	}
};

export const logErr = (exception: unknown): void => {
	if (Object.hasOwn(exception as object, "name")) {
		if ((exception as Error).name !== "AbortError") {
			return log(exception, Level.err);
		}
	}
	return log(exception, Level.err);
};
