import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import GroupsIcon from '@mui/icons-material/Groups';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import MenuItem from '@mui/material/MenuItem';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { parseISO } from 'date-fns';
import { z } from 'zod';
import { apiCreateBanGroup, apiUpdateBanGroup, Duration, DurationCollection, GroupBanRecord } from '../../api';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { makeValidateSteamIDCallback } from '../../util/validator/makeValidateSteamIDCallback.ts';
import { Heading } from '../Heading';
import { Buttons } from '../field/Buttons.tsx';
import { DateTimeSimple } from '../field/DateTimeSimple.tsx';
import { MarkdownField } from '../field/MarkdownField.tsx';
import { SelectFieldSimple } from '../field/SelectFieldSimple.tsx';
import { SteamIDField } from '../field/SteamIDField.tsx';
import { TextFieldSimple } from '../field/TextFieldSimple.tsx';

type BanGroupFormValues = {
    ban_group_id?: number;
    target_id: string;
    group_id: string;
    duration: Duration;
    duration_custom: string;
    note: string;
};

export const BanGroupModal = NiceModal.create(({ existing }: { existing?: GroupBanRecord }) => {
    const modal = useModal();
    const { sendFlash } = useUserFlashCtx();

    const mutation = useMutation({
        mutationKey: ['banGroup'],
        mutationFn: async (values: BanGroupFormValues) => {
            try {
                if (existing?.ban_group_id) {
                    const ban_record = apiUpdateBanGroup(existing.ban_group_id, {
                        note: values.note,
                        target_id: values.target_id,
                        valid_until: values.duration_custom ? parseISO(values.duration_custom) : undefined
                    });
                    sendFlash('success', 'Updated group ban successfully');
                    modal.resolve(ban_record);
                } else {
                    const ban_record = await apiCreateBanGroup({
                        note: values.note,
                        duration: values.duration,
                        valid_until: values.duration_custom ? parseISO(values.duration_custom) : undefined,
                        target_id: values.target_id,
                        group_id: values.group_id
                    });
                    sendFlash('success', 'Created group ban successfully');
                    modal.resolve(ban_record);
                }
                await modal.hide();
            } catch (e) {
                modal.reject(e);
            }
        }
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutation.mutate({
                target_id: value.target_id,
                group_id: value.group_id,
                duration: value.duration,
                duration_custom: value.duration_custom,
                note: value.note
            });
        },
        validators: {
            onChangeAsyncDebounceMs: 500,
            onChangeAsync: z.object({
                target_id: makeValidateSteamIDCallback(),
                group_id: z.string(),
                duration: z.nativeEnum(Duration),
                duration_custom: z.string(),
                note: z.string()
            })
        },
        defaultValues: {
            target_id: existing ? existing.target_id : '',
            group_id: existing ? existing.group_id : '',
            duration: existing ? Duration.durCustom : Duration.dur2w,
            duration_custom: existing ? existing.valid_until.toISOString() : '',
            note: existing ? existing.note : ''
        }
    });
    return (
        <Dialog fullWidth {...muiDialogV5(modal)}>
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await handleSubmit();
                }}
            >
                <DialogTitle component={Heading} iconLeft={<GroupsIcon />}>
                    Ban Steam Group
                </DialogTitle>

                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid xs={12}>
                            <Field
                                name={'target_id'}
                                children={(props) => {
                                    return (
                                        <SteamIDField
                                            {...props}
                                            defaultValue={props.state.value}
                                            label={'Target Steam ID'}
                                            fullwidth={true}
                                            disabled={Boolean(existing?.ban_group_id)}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid xs={12}>
                            <Field
                                name={'group_id'}
                                children={(props) => {
                                    return (
                                        <TextFieldSimple
                                            {...props}
                                            value={props.state.value}
                                            label={'Steam Group ID'}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'duration'}
                                children={(props) => {
                                    return (
                                        <SelectFieldSimple
                                            {...props}
                                            label={'Duration'}
                                            fullwidth={true}
                                            defaultValue={props.state.value}
                                            items={DurationCollection}
                                            renderMenu={(du) => {
                                                return (
                                                    <MenuItem value={du} key={`du-${du}`}>
                                                        {du}
                                                    </MenuItem>
                                                );
                                            }}
                                        />
                                    );
                                }}
                            />
                        </Grid>

                        <Grid xs={6}>
                            <Field
                                name={'duration_custom'}
                                children={(props) => {
                                    return (
                                        <DateTimeSimple
                                            {...props}
                                            defaultValue={props.state.value}
                                            label={'Custom Expire Date'}
                                        />
                                    );
                                }}
                            />
                        </Grid>

                        <Grid xs={12}>
                            <Field
                                name={'note'}
                                children={(props) => {
                                    return (
                                        <MarkdownField
                                            {...props}
                                            defaultValue={props.state.value}
                                            multiline={true}
                                            rows={10}
                                            label={'Mod Notes'}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                    </Grid>
                </DialogContent>
                <DialogActions>
                    <Grid container>
                        <Grid xs={12} mdOffset="auto">
                            <Subscribe
                                selector={(state) => [state.canSubmit, state.isSubmitting]}
                                children={([canSubmit, isSubmitting]) => {
                                    return (
                                        <Buttons
                                            reset={reset}
                                            canSubmit={canSubmit}
                                            isSubmitting={isSubmitting}
                                            onClose={async () => {
                                                await modal.hide();
                                            }}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                    </Grid>
                </DialogActions>
            </form>
        </Dialog>
    );
});
