import type { ReactNode } from "react";

export const renderHelpText = (errors: unknown[], helpText?: string | ReactNode) => {
	return errors.length > 0
		? errors
				.map((v: unknown) => {
					try {
						// @ts-expect-error fix later
						return v.message;
					} catch {
						return String(v);
					}
				})
				.join(", ")
		: helpText;
};
