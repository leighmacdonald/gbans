import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import MenuItem from '@mui/material/MenuItem';
import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';
import { apiCreateFilter, apiEditFilter } from '../../api/filters.ts';
import { useAppForm } from '../../contexts/formContext.tsx';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import {
    Filter,
    FilterAction,
    FilterActionCollection,
    FilterActionEnum,
    filterActionString
} from '../../schema/filters.ts';
import { Heading } from '../Heading';

type FilterEditFormValues = {
    pattern: string;
    is_regex: boolean;
    is_enabled?: boolean;
    action: FilterActionEnum;
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

    const form = useAppForm({
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
                    await form.handleSubmit();
                }}
            >
                <DialogTitle component={Heading} iconLeft={<FilterAltIcon />}>
                    Filter Editor
                </DialogTitle>
                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 8 }}>
                            <form.AppField
                                name={'pattern'}
                                children={(field) => {
                                    return <field.TextField label={'Pattern'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 4 }}>
                            <form.AppField
                                name={'is_regex'}
                                children={(field) => {
                                    return <field.CheckboxField label={'Is Regex Pattern'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 4 }}>
                            <form.AppField
                                name={'action'}
                                children={(field) => {
                                    return (
                                        <field.SelectField
                                            label={'Action'}
                                            items={FilterActionCollection}
                                            renderItem={(fa) => {
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
                            <form.AppField
                                name={'duration'}
                                children={(field) => {
                                    return <field.TextField label={'Duration'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 4 }}>
                            <form.AppField
                                name={'weight'}
                                children={(field) => {
                                    return <field.TextField label={'Weight (1-100)'} />;
                                }}
                            />
                        </Grid>

                        <Grid size={{ xs: 4 }}>
                            <form.AppField
                                name={'is_enabled'}
                                validators={{
                                    onSubmit: z.boolean()
                                }}
                                children={(field) => {
                                    return <field.CheckboxField label={'Is Enabled'} />;
                                }}
                            />
                        </Grid>
                    </Grid>
                </DialogContent>
                <DialogActions>
                    <Grid container>
                        <Grid size={{ xs: 12 }}>
                            <form.AppForm>
                                <ButtonGroup>
                                    <form.ResetButton />
                                    <form.SubmitButton />
                                </ButtonGroup>
                            </form.AppForm>
                        </Grid>
                    </Grid>
                </DialogActions>
            </form>
        </Dialog>
    );
});
