import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import PersonIcon from '@mui/icons-material/Person';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Grid from '@mui/material/Grid2';
import MenuItem from '@mui/material/MenuItem';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';
import {
    apiUpdatePlayerPermission,
    PermissionLevel,
    PermissionLevelCollection,
    permissionLevelString,
    Person
} from '../../api';
import { Heading } from '../Heading';
import { Buttons } from '../field/Buttons.tsx';
import { SelectFieldSimple } from '../field/SelectFieldSimple.tsx';

export const PersonEditModal = NiceModal.create(({ person }: { person: Person }) => {
    const modal = useModal();

    const mutation = useMutation({
        mutationKey: ['banCIDR'],
        mutationFn: async (values: { permission_level: PermissionLevel }) => {
            try {
                const updatedPerson = await apiUpdatePlayerPermission(person.steam_id, {
                    permission_level: values.permission_level
                });
                modal.resolve(updatedPerson);
            } catch (e) {
                modal.reject(e);
            }
            await modal.hide();
        }
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutation.mutate({
                permission_level: value.permission_level
            });
        },
        validators: {
            onChange: z.object({
                permission_level: z.nativeEnum(PermissionLevel)
            })
        },
        defaultValues: {
            permission_level: person.permission_level
        }
    });

    return (
        <Dialog {...muiDialogV5(modal)} fullWidth maxWidth={'sm'}>
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await handleSubmit();
                }}
            >
                <DialogTitle component={Heading} iconLeft={<PersonIcon />}>
                    Person Editor: {person.personaname}
                </DialogTitle>
                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 12 }}>
                            <Field
                                name={'permission_level'}
                                children={(props) => {
                                    return (
                                        <SelectFieldSimple
                                            {...props}
                                            label={'Permissions'}
                                            fullwidth={true}
                                            items={PermissionLevelCollection}
                                            renderMenu={(pl) => {
                                                return (
                                                    <MenuItem value={pl} key={`pl-${pl}`}>
                                                        {permissionLevelString(pl)}
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
