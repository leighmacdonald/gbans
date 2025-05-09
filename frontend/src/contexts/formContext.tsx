import { createFormHook, createFormHookContexts } from '@tanstack/react-form';
import { CheckboxField } from '../component/field/CheckboxField.tsx';
import { ClearButton } from '../component/field/ClearButton.tsx';
import { CloseButton } from '../component/field/CloseButton.tsx';
import { DateTimeField } from '../component/field/DateTimeField.tsx';
import { MarkdownField } from '../component/field/MarkdownField.tsx';
import { ResetButton } from '../component/field/ResetButton.tsx';
import { SelectField } from '../component/field/SelectField.tsx';
import { SteamIDField } from '../component/field/SteamIDField.tsx';
import { SubmitButton } from '../component/field/SubmitButton.tsx';
import { TextField } from '../component/field/TextField.tsx';

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
        DateTimeField
    },
    formComponents: {
        SubmitButton,
        ClearButton,
        ResetButton,
        CloseButton
    }
});
