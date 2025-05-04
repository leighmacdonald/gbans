import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Grid from '@mui/material/Grid';
import MenuItem from '@mui/material/MenuItem';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';
import {
    apiCreateFilter,
    apiEditFilter,
    Filter,
    FilterAction,
    FilterActionCollection,
    filterActionString
} from '../../api/filters.ts';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { Heading } from '../Heading';
import { Buttons } from '../field/Buttons.tsx';
import { CheckboxSimple } from '../field/CheckboxSimple.tsx';
import { SelectFieldSimple } from '../field/SelectFieldSimple.tsx';
import { TextFieldSimple } from '../field/TextFieldSimple.tsx';

type FilterEditFormValues = {
    pattern: string;
    is_regex: boolean;
    is_enabled?: boolean;
    action: FilterAction;
    duration: string;
    weight: string;
};

export const FilterEditModal = NiceModal.create(({ filter }: { filter?: Filter }) => {
    const modal = useModal();
    const { sendError } = useUserFlashCtx();

    const mutation = useMutation({
        mutationKey: ['filters'],
        mutationFn: async (values: FilterEditFormValues) => {
            if (filter?.filter_id) {
                return await apiEditFilter(filter?.filter_id, {
                    is_enabled: values.is_enabled,
                    is_regex: values.is_regex,
                    pattern: values.pattern,
                    action: values.action,
                    duration: values.duration,
                    weight: Number(values.weight)
                });
            } else {
                return await apiCreateFilter({
                    is_enabled: values.is_enabled,
                    is_regex: values.is_regex,
                    pattern: values.pattern,
                    action: values.action,
                    duration: values.duration,
                    weight: Number(values.weight)
                });
            }
        },
        onSuccess: async (result) => {
            modal.resolve(result);
            await modal.hide();
        },
        onError: sendError
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutation.mutate({
                pattern: String(value.pattern),
                action: value.action,
                duration: value.duration,
                weight: value.weight,
                is_enabled: value.is_enabled,
                is_regex: value.is_regex
            });
        },
        defaultValues: {
            pattern: filter ? String(filter.pattern) : '',
            is_regex: filter?.is_regex ?? false,
            is_enabled: filter?.is_enabled ?? true,
            action: filter?.action ?? FilterAction.Kick,
            duration: filter?.duration ?? '1w',
            weight: filter ? String(filter.weight) : '1'
        },
        validators: {
            onSubmit: z.object({
                pattern: z.string({ message: 'Must entry pattern' }).min(2),
                is_regex: z.boolean(),
                action: z.nativeEnum(FilterAction, { message: 'Must select an action' }),
                duration: z.string({ message: 'Must provide a duration' }),
                weight: z.string(),
                is_enabled: z.boolean()
            })
        }
    });

    return (
        <Dialog {...muiDialogV5(modal)} fullWidth maxWidth={'md'}>
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await handleSubmit();
                }}
            >
                <DialogTitle component={Heading} iconLeft={<FilterAltIcon />}>
                    Filter Editor
                </DialogTitle>
                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 8 }}>
                            <Field
                                name={'pattern'}
                                children={(props) => {
                                    return <TextFieldSimple {...props} value={props.state.value} label={'Pattern'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 4 }}>
                            <Field
                                name={'is_regex'}
                                children={({ state, handleBlur, handleChange }) => {
                                    return (
                                        <CheckboxSimple
                                            value={state.value}
                                            onBlur={handleBlur}
                                            onChange={(_, v) => {
                                                handleChange(v);
                                            }}
                                            label={'Is Regex Pattern'}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 4 }}>
                            <Field
                                name={'action'}
                                children={(props) => {
                                    return (
                                        <SelectFieldSimple
                                            {...props}
                                            value={props.state.value}
                                            label={'Action'}
                                            items={FilterActionCollection}
                                            renderMenu={(fa) => {
                                                return (
                                                    <MenuItem value={fa} key={`fa-${fa}`}>
                                                        {filterActionString(fa)}
                                                    </MenuItem>
                                                );
                                            }}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 4 }}>
                            <Field
                                name={'duration'}
                                children={(props) => {
                                    return <TextFieldSimple {...props} value={props.state.value} label={'Duration'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 4 }}>
                            <Field
                                name={'weight'}
                                children={(props) => {
                                    return (
                                        <TextFieldSimple
                                            {...props}
                                            value={props.state.value}
                                            label={'Weight (1-100)'}
                                        />
                                    );
                                }}
                            />
                        </Grid>

                        <Grid size={{ xs: 4 }}>
                            <Field
                                name={'is_enabled'}
                                validators={{
                                    onSubmit: z.boolean()
                                }}
                                children={({ state, handleBlur, handleChange }) => {
                                    return (
                                        <CheckboxSimple
                                            value={state.value}
                                            onBlur={handleBlur}
                                            onChange={(_, v) => {
                                                handleChange(v);
                                            }}
                                            label={'Is Enabled'}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                    </Grid>
                </DialogContent>
                <DialogActions>
                    <Grid container>
                        <Grid size={{ xs: 12 }}>
                            <Subscribe
                                selector={(state) => [state.canSubmit, state.isSubmitting]}
                                children={([canSubmit, isSubmitting]) => {
                                    return <Buttons reset={reset} canSubmit={canSubmit} isSubmitting={isSubmitting} />;
                                }}
                            />
                        </Grid>
                    </Grid>
                </DialogActions>
            </form>
        </Dialog>
    );
});
