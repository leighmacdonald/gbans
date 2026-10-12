import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { describe, expect, test } from "vitest";
import { friendlyError } from "./error.tsx";
import { ViolationsSchema } from "./rpc/buf/validate/validate_pb.ts";

const violations = create(ViolationsSchema, {
	violations: [
		{
			message: "value is required",
			field: { elements: [{ fieldName: "target_id" }] },
		},
		{
			message: "value length must be at least 2",
			field: { elements: [{ fieldName: "reason_text" }] },
		},
	],
});

const connectError = (
	message: string,
	code: Code,
	details: { desc: typeof ViolationsSchema; value: typeof violations }[] = [],
) => new ConnectError(message, code, undefined, details, undefined);

describe("friendlyError", () => {
	test("rewrites validation violations as plain per-field messages", () => {
		const err = connectError(
			"target_id: value is required [required], and 1 more violation",
			Code.InvalidArgument,
			[{ desc: ViolationsSchema, value: violations }],
		);
		const { heading, message } = friendlyError(err);
		expect(heading).toBe("Invalid input");
		expect(message).toBe("Target ID: value is required; Reason Text: value length must be at least 2");
		expect(message).not.toContain("target_id");
		expect(message).not.toContain("[required]");
	});

	test("invalid_argument without details falls back to the raw message", () => {
		const err = connectError("something failed", Code.InvalidArgument);
		const { heading, message } = friendlyError(err);
		expect(heading).toBe("Invalid input");
		expect(message).toBe("something failed");
	});

	test("maps well-known codes to friendly headings", () => {
		expect(friendlyError(connectError("denied", Code.PermissionDenied)).heading).toBe("Permission denied");
		expect(friendlyError(connectError("missing", Code.Unauthenticated)).heading).toBe("Login required");
		expect(friendlyError(connectError("gone", Code.NotFound)).heading).toBe("Not found");
	});

	test("passes plain errors through unchanged", () => {
		const err = new TypeError("boom");
		const { heading, message } = friendlyError(err);
		expect(heading).toBe("TypeError");
		expect(message).toBe("boom");
	});
});
