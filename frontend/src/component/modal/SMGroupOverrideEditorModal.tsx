import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import GroupsIcon from '@mui/icons-material/Groups';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import MenuItem from '@mui/material/MenuItem';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { zodValidator } from '@tanstack/zod-form-adapter';
import 'video-react/dist/video-react.css';
import {
    apiCreateSMGroupOverrides,
    apiSaveSMGroupOverrides,
    OverrideAccess,
    OverrideType,
    SMGroupOverrides,
    SMGroups
} from '../../api';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { Heading } from '../Heading';
import { Buttons } from '../field/Buttons.tsx';
import { SelectFieldSimple } from '../field/SelectFieldSimple.tsx';
import { TextFieldSimple } from '../field/TextFieldSimple.tsx';

type mutateOverrideArgs = {
    name: string;
    type: OverrideType;
    access: OverrideAccess;
};

export const SMGroupOverrideEditorModal = NiceModal.create(
    ({ group, override }: { group: SMGroups; override?: SMGroupOverrides }) => {
        const modal = useModal();
        const { sendFlash } = useUserFlashCtx();
        const mutation = useMutation({
            mutationKey: ['adminSMGroupOverride'],
            mutationFn: async ({ name, type, access }: mutateOverrideArgs) => {
                return override?.group_override_id
                    ? await apiSaveSMGroupOverrides(override.group_override_id, name, type, access)
                    : await apiCreateSMGroupOverrides(group.group_id, name, type, access);
            },
            onSuccess: async (override) => {
                modal.resolve(override);
                await modal.hide();
            },
            onError: (error) => {
                sendFlash('error', `Failed to create group override: ${error}`);
            }
        });

        const { Field, Subscribe, handleSubmit, reset } = useForm({
            onSubmit: async ({ value }) => {
                mutation.mutate(value);
            },
            validatorAdapter: zodValidator,
            defaultValues: {
                type: override?.type ?? 'command',
                name: override?.name ?? '',
                access: override?.access ?? 'allow'
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
                        {override ? 'Edit' : 'Create'} Group Override
                    </DialogTitle>

                    <DialogContent>
                        <Grid container spacing={2}>
                            <Grid xs={6}>
                                <Field
                                    name={'name'}
                                    children={(props) => {
                                        return <TextFieldSimple {...props} label={'Name'} fullwidth={true} />;
                                    }}
                                />
                            </Grid>
                            <Grid xs={6}>
                                <Field
                                    name={'type'}
                                    children={(props) => {
                                        return (
                                            <SelectFieldSimple
                                                {...props}
                                                label={'Override Type'}
                                                fullwidth={true}
                                                items={['command', 'group']}
                                                renderMenu={(i) => {
                                                    return (
                                                        <MenuItem value={i} key={i}>
                                                            {i}
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
                                    name={'access'}
                                    children={(props) => {
                                        return (
                                            <SelectFieldSimple
                                                {...props}
                                                label={'Access Type'}
                                                fullwidth={true}
                                                items={['allow', 'deny']}
                                                renderMenu={(i) => {
                                                    return (
                                                        <MenuItem value={i} key={i}>
                                                            {i}
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
                                                submitLabel={'Submit'}
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
    }
);
