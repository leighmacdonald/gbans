import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import PersonIcon from '@mui/icons-material/Person';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import MenuItem from '@mui/material/MenuItem';
import { useMutation } from '@tanstack/react-query';
import {
    apiUpdatePlayerPermission,
    PermissionLevel,
    PermissionLevelCollection,
    permissionLevelString,
    Person
} from '../../api';
import { useAppForm } from '../../contexts/formContext.tsx';
import { Heading } from '../Heading';

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

    const form = useAppForm({
        onSubmit: async ({ value }) => {
            mutation.mutate({
                permission_level: value.permission_level
            });
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
                    await form.handleSubmit();
                }}
            >
                <DialogTitle component={Heading} iconLeft={<PersonIcon />}>
                    Person Editor: {person.personaname}
                </DialogTitle>
                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'permission_level'}
                                children={(field) => {
                                    return (
                                        <field.SelectField
                                            label={'Permissions'}
                                            items={PermissionLevelCollection}
                                            renderItem={(pl) => {
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
