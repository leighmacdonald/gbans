import { useForm } from "@tanstack/react-form";
import type { ReactNode } from "react";
import { renderToString } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { formContext } from "../../../contexts/formContext.tsx";
import { CloseButton } from "./CloseButton.tsx";

const WithForm = ({ children }: { children: ReactNode }) => {
	const formApi = useForm({ defaultValues: { bodyMd: "" } });
	return <formContext.Provider value={formApi}>{children}</formContext.Provider>;
};

describe("CloseButton", () => {
	it("renders outside a NiceModal context when a custom onClick is provided (eg. inline wiki edit mode)", () => {
		const html = renderToString(
			<WithForm>
				<CloseButton onClick={() => {}} />
			</WithForm>,
		);
		expect(html).toContain("Close");
	});

	it("throws outside a NiceModal context only when the modal close action is required", () => {
		expect(() =>
			renderToString(
				<WithForm>
					<CloseButton />
				</WithForm>,
			),
		).toThrow(/no modal id found/i);
	});
});
