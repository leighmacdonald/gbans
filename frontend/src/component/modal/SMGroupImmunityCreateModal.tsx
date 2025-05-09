import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import GroupsIcon from '@mui/icons-material/Groups';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Grid from '@mui/material/Grid';
import MenuItem from '@mui/material/MenuItem';
import { useMutation } from '@tanstack/react-query';
import 'video-react/dist/video-react.css';
import { apiCreateSMGroupImmunity, SMGroups } from '../../api';
import { useAppForm } from '../../contexts/formContext.tsx';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { Heading } from '../Heading';

export const SMGroupImmunityCreateModal = NiceModal.create(({ groups }: { groups: SMGroups[] }) => {
    const modal = useModal();
    const { sendError } = useUserFlashCtx();

    const mutation = useMutation({
        mutationKey: ['createGroupImmunity'],
        mutationFn: async ({ group, other }: { group: SMGroups; other: SMGroups }) => {
            // FIXME How to get number from select properly typed?
            return await apiCreateSMGroupImmunity(group as unknown as number, other as unknown as number);
        },
        onSuccess: async (immunity) => {
            modal.resolve(immunity);
            await modal.hide();
        },
        onError: sendError
    });

    const form = useAppForm({
        onSubmit: async ({ value }) => {
            mutation.mutate(value);
        },
        defaultValues: {
            group: groups[0],
            other: groups[1]
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
                    Select Group
                </DialogTitle>

                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'group'}
                                children={(field) => {
                                    return (
                                        <field.SelectField
                                            label={'Group'}
                                            items={groups}
                                            renderItem={(i) => {
                                                if (!i) {
                                                    return;
                                                }
                                                return (
                                                    <MenuItem value={i.group_id} key={i.group_id}>
                                                        {i.name}
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
                                name={'other'}
                                children={(field) => {
                                    return (
                                        <field.SelectField
                                            label={'Immunity From'}
                                            items={groups}
                                            renderItem={(i) => {
                                                if (!i) {
                                                    return;
                                                }
                                                return (
                                                    <MenuItem value={i.group_id} key={i.group_id}>
                                                        {i.name}
                                                    </MenuItem>
                                                );
                                            }}
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
                            <form.AppForm>
                                <form.CloseButton
                                    onClick={async () => {
                                        await modal.hide();
                                    }}
                                />
                                <form.ResetButton />
                                <form.SubmitButton />
                            </form.AppForm>
                        </Grid>
                    </Grid>
                </DialogActions>
            </form>
        </Dialog>
    );
});
