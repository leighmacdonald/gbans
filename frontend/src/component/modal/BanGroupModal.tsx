import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import GroupsIcon from '@mui/icons-material/Groups';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import MenuItem from '@mui/material/MenuItem';
import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';
import { apiCreateBanGroup, apiUpdateBanGroup, Duration, DurationCollection, GroupBanRecord } from '../../api';
import { useAppForm } from '../../contexts/formContext.tsx';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { Heading } from '../Heading';

const schema = z.object({
    target_id: z.string(),
    group_id: z.string(),
    duration: z.nativeEnum(Duration),
    duration_custom: z.date(),
    note: z.string()
});

export const BanGroupModal = NiceModal.create(({ existing }: { existing?: GroupBanRecord }) => {
    const modal = useModal();
    const { sendFlash } = useUserFlashCtx();

    const mutation = useMutation({
        mutationKey: ['banGroup'],
        mutationFn: async (values: z.input<typeof schema>) => {
            try {
                if (existing?.ban_group_id) {
                    const ban_record = apiUpdateBanGroup(existing.ban_group_id, {
                        note: values.note,
                        target_id: values.target_id,
                        valid_until: values.duration_custom
                    });
                    sendFlash('success', 'Updated CIDR ban successfully');
                    modal.resolve(ban_record);
                } else {
                    const ban_record = await apiCreateBanGroup({
                        note: values.note,
                        duration: values.duration,
                        valid_until: values.duration_custom,
                        target_id: values.target_id,
                        group_id: values.group_id
                    });
                    sendFlash('success', 'Created CIDR ban successfully');
                    modal.resolve(ban_record);
                }
                await modal.hide();
            } catch (e) {
                modal.reject(e);
            }
        }
    });

    const form = useAppForm({
        onSubmit: async ({ value }) => {
            mutation.mutate({
                target_id: value.target_id,
                group_id: value.group_id,
                duration: value.duration,
                duration_custom: value.duration_custom,
                note: value.note
            });
        },
        defaultValues: {
            target_id: existing ? existing.target_id : '',
            group_id: existing ? existing.group_id : '',
            duration: existing ? Duration.durCustom : Duration.dur2w,
            duration_custom: existing?.valid_until ?? new Date(),
            note: existing ? existing.note : ''
        },
        validators: {
            onSubmit: schema
        }
    });
    return (
        <Dialog fullWidth {...muiDialogV5(modal)}>
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await form.handleSubmit();
                }}
            >
                <DialogTitle component={Heading} iconLeft={<GroupsIcon />}>
                    Ban Steam Group
                </DialogTitle>

                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'target_id'}
                                children={(field) => {
                                    return (
                                        <field.SteamIDField
                                            label={'Target Steam ID'}
                                            disabled={Boolean(existing?.ban_group_id)}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'group_id'}
                                children={(field) => {
                                    return <field.TextField label={'Steam Group ID'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'duration'}
                                children={(field) => {
                                    return (
                                        <field.SelectField
                                            label={'Duration'}
                                            items={DurationCollection}
                                            renderItem={(du) => {
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

                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'duration_custom'}
                                children={(field) => {
                                    return <field.DateTimeField label={'Custom Expire Date'} />;
                                }}
                            />
                        </Grid>

                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'note'}
                                children={(field) => {
                                    return <field.MarkdownField multiline={true} rows={10} label={'Mod Notes'} />;
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
