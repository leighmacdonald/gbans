import { createFormHook, createFormHookContexts } from "@tanstack/react-form";
import { lazy } from "react";

const ClearButton = lazy(() => import("../component/form/button/ClearButton.tsx"));
const CloseButton = lazy(() => import("../component/form/button/CloseButton.tsx"));
const ResetButton = lazy(() => import("../component/form/button/ResetButton.tsx"));
const SubmitButton = lazy(() => import("../component/form/button/SubmitButton.tsx"));
const CheckboxField = lazy(() => import("../component/form/field/CheckboxField.tsx"));
const DateTimeField = lazy(() => import("../component/form/field/DateTimeField.tsx"));
const MarkdownField = lazy(() => import("../component/form/field/MarkdownField.tsx"));
const NumberField = lazy(() => import("../component/form/field/NumberField.tsx"));
const SelectField = lazy(() => import("../component/form/field/SelectField.tsx"));
const SteamIDField = lazy(() => import("../component/form/field/SteamIDField.tsx"));
const TextField = lazy(() => import("../component/form/field/TextField.tsx"));

export const { fieldContext, formContext, useFieldContext, useFormContext } = createFormHookContexts();

export const { useAppForm, withForm } = createFormHook({
	fieldContext,
	formContext,
	fieldComponents: {
		CheckboxField,
		TextField,
		SteamIDField,
		SelectField,
		DateTimeField,
		NumberField,
		MarkdownField,
	},
	formComponents: {
		SubmitButton,
		ClearButton,
		ResetButton,
		CloseButton,
	},
});
