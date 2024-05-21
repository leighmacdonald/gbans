import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import GroupsIcon from '@mui/icons-material/Groups';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import MenuItem from '@mui/material/MenuItem';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import 'video-react/dist/video-react.css';
import { apiCreateSMGroupImmunity, SMGroups } from '../../api';
import { Heading } from '../Heading';
import { Buttons } from '../field/Buttons.tsx';
import { SelectFieldSimple } from '../field/SelectFieldSimple.tsx';

export const SMGroupImmunityCreateModal = NiceModal.create(({ groups }: { groups: SMGroups[] }) => {
    const modal = useModal();

    const mutation = useMutation({
        mutationKey: ['createGroupImmunity'],
        mutationFn: async ({ group, other }: { group: SMGroups; other: SMGroups }) => {
            return await apiCreateSMGroupImmunity(group.group_id, other.group_id);
        },
        onSuccess: async (immunity) => {
            modal.resolve(immunity);
            await modal.hide();
        }
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm({
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
                    await handleSubmit();
                }}
            >
                <DialogTitle component={Heading} iconLeft={<GroupsIcon />}>
                    Select Group
                </DialogTitle>

                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid xs={6}>
                            <Field
                                name={'group'}
                                children={(props) => {
                                    return (
                                        <SelectFieldSimple
                                            {...props}
                                            label={'Group'}
                                            fullwidth={true}
                                            items={groups}
                                            renderMenu={(i) => {
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
                        <Grid xs={6}>
                            <Field
                                name={'other'}
                                children={(props) => {
                                    return (
                                        <SelectFieldSimple
                                            {...props}
                                            label={'Immunity From'}
                                            fullwidth={true}
                                            items={groups}
                                            renderMenu={(i) => {
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
                        <Grid xs={12} mdOffset="auto">
                            <Subscribe
                                selector={(state) => [state.canSubmit, state.isSubmitting]}
                                children={([canSubmit, isSubmitting]) => {
                                    return (
                                        <Buttons
                                            reset={reset}
                                            canSubmit={canSubmit}
                                            showReset={false}
                                            submitLabel={'Select Group'}
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
