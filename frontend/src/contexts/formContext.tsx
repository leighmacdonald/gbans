import { createFormHook, createFormHookContexts } from "@tanstack/react-form";
import { ClearButton } from "../component/form/button/ClearButton.tsx";
import { CloseButton } from "../component/form/button/CloseButton.tsx";
import { ResetButton } from "../component/form/button/ResetButton.tsx";
import { SubmitButton } from "../component/form/button/SubmitButton.tsx";
import { CheckboxField } from "../component/form/field/CheckboxField.tsx";
import { DateTimeField } from "../component/form/field/DateTimeField.tsx";
import { MarkdownField } from "../component/form/field/MarkdownField.tsx";
import { NumberField } from "../component/form/field/NumberField.tsx";
import { SelectField } from "../component/form/field/SelectField.tsx";
import { SteamIDField } from "../component/form/field/SteamIDField.tsx";
import { TextField } from "../component/form/field/TextField.tsx";

export const { fieldContext, formContext, useFieldContext, useFormContext } = createFormHookContexts();

export const { useAppForm } = createFormHook({
	fieldContext,
	formContext,
	fieldComponents: {
		CheckboxField,
		MarkdownField,
		TextField,
		SteamIDField,
		SelectField,
		DateTimeField,
		NumberField,
	},
	formComponents: {
		SubmitButton,
		ClearButton,
		ResetButton,
		CloseButton,
	},
});
