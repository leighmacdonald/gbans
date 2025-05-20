import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import GroupsIcon from '@mui/icons-material/Groups';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import { useMutation } from '@tanstack/react-query';
import 'video-react/dist/video-react.css';
import { z } from 'zod/v4';
import { apiCreateSMGroup, apiSaveSMGroup, hasSMFlag } from '../../api';
import { useAppForm } from '../../contexts/formContext.tsx';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { schemaFlags, SMGroups } from '../../schema/sourcemod.ts';
import { Heading } from '../Heading';

const schema = schemaFlags.extend({
    name: z.string().min(2),
    immunity: z.number().min(0).max(100)
});

export const SMGroupEditorModal = NiceModal.create(({ group }: { group?: SMGroups }) => {
    const modal = useModal();
    const { sendError } = useUserFlashCtx();
    const defaultValues: z.input<typeof schema> = {
        name: group?.name ?? '',
        immunity: group?.immunity_level ?? 0,
        z: hasSMFlag('z', group),
        a: hasSMFlag('a', group),
        b: hasSMFlag('b', group),
        c: hasSMFlag('c', group),
        d: hasSMFlag('d', group),
        e: hasSMFlag('e', group),
        f: hasSMFlag('f', group),
        g: hasSMFlag('g', group),
        h: hasSMFlag('h', group),
        i: hasSMFlag('i', group),
        j: hasSMFlag('j', group),
        k: hasSMFlag('k', group),
        l: hasSMFlag('l', group),
        m: hasSMFlag('m', group),
        n: hasSMFlag('n', group),
        o: hasSMFlag('o', group),
        p: hasSMFlag('p', group),
        q: hasSMFlag('q', group),
        r: hasSMFlag('r', group),
        s: hasSMFlag('s', group),
        t: hasSMFlag('t', group)
    };
    const edit = useMutation({
        mutationKey: ['adminSMGroup'],
        mutationFn: async ({ name, immunity, flags }: { name: string; immunity: number; flags: string }) => {
            if (group?.group_id) {
                return await apiSaveSMGroup(group.group_id, name, immunity, flags);
            }
            return await apiCreateSMGroup(name, immunity, flags);
        },
        onSuccess: async (group) => {
            modal.resolve(group);
            await modal.hide();
        },
        onError: sendError
    });

    const form = useAppForm({
        onSubmit: async ({ value }) => {
            const flags = Object.entries(value)
                .filter((v) => !['name', 'immunity'].includes(v[0]))
                .reduce((acc, value) => {
                    if (value[1]) {
                        acc += value[0];
                    }
                    return acc;
                }, '');
            edit.mutate({ name: value.name, immunity: Number(value.immunity), flags: flags });
        },
        validators: {
            onSubmit: schema
        },
        defaultValues
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
                    SM Group Editor
                </DialogTitle>

                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'name'}
                                children={(field) => {
                                    return <field.TextField label={'Group Name'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'immunity'}
                                children={(field) => {
                                    return <field.TextField label={'Immunity Level'} />;
                                }}
                            />
                        </Grid>

                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'z'}
                                children={(field) => {
                                    return <field.CheckboxField label={'(z) Full Admin'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'a'}
                                children={(field) => {
                                    return <field.CheckboxField label={'(a) Reserved Slot'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'b'}
                                children={(field) => {
                                    return <field.CheckboxField label={'(b) Generic Admin'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'c'}
                                children={(field) => {
                                    return <field.CheckboxField label={'(c) Kick Players'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'d'}
                                children={(field) => {
                                    return <field.CheckboxField label={'(d) Ban Players'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'e'}
                                children={(field) => {
                                    return <field.CheckboxField label={'(e) Unban Players'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'f'}
                                children={(field) => {
                                    return <field.CheckboxField label={'(f) Slay/Harm Players'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'g'}
                                children={(field) => {
                                    return <field.CheckboxField label={'(g) Change Maps'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'h'}
                                children={(field) => {
                                    return <field.CheckboxField label={'(h) Change CVARs'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'i'}
                                children={(field) => {
                                    return <field.CheckboxField label={'(i) Exec Configs'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'j'}
                                children={(field) => {
                                    return <field.CheckboxField label={'(j) Special Chat Privileges'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'k'}
                                children={(field) => {
                                    return <field.CheckboxField label={'(k) Start Votes'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'l'}
                                children={(field) => {
                                    return <field.CheckboxField label={'(l) Set Server Password'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'m'}
                                children={(field) => {
                                    return <field.CheckboxField label={'(m) RCON Access'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'n'}
                                children={(field) => {
                                    return <field.CheckboxField label={'(n) Enabled Cheats'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'o'}
                                children={(field) => {
                                    return <field.CheckboxField label={'(o) Custom Flag'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'p'}
                                children={(field) => {
                                    return <field.CheckboxField label={'(p) Custom Flag'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'q'}
                                children={(field) => {
                                    return <field.CheckboxField label={'(q) Custom Flag'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'r'}
                                children={(field) => {
                                    return <field.CheckboxField label={'(r) Custom Flag'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'s'}
                                children={(field) => {
                                    return <field.CheckboxField label={'(s) Custom Flag'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'t'}
                                children={(field) => {
                                    return <field.CheckboxField label={'(t) Custom Flag'} />;
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
