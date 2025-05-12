import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import GroupsIcon from '@mui/icons-material/Groups';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import MenuItem from '@mui/material/MenuItem';
import { useMutation } from '@tanstack/react-query';
import 'video-react/dist/video-react.css';
import { z } from 'zod';
import { apiCreateSMOverrides, apiSaveSMOverrides, hasSMFlag, OverrideType, SMOverrides } from '../../api';
import { useAppForm } from '../../contexts/formContext.tsx';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { Heading } from '../Heading';

type mutateOverrideArgs = {
    name: string;
    type: OverrideType;
    flags: string;
};

export const SMOverrideEditorModal = NiceModal.create(({ override }: { override?: SMOverrides }) => {
    const modal = useModal();
    const { sendError } = useUserFlashCtx();
    const mutation = useMutation({
        mutationKey: ['adminSMOverride'],
        mutationFn: async ({ name, type, flags }: mutateOverrideArgs) => {
            return override?.override_id
                ? await apiSaveSMOverrides(override.override_id, name, type, flags)
                : await apiCreateSMOverrides(name, type, flags);
        },
        onSuccess: async (admin) => {
            modal.resolve(admin);
            await modal.hide();
        },
        onError: sendError
    });

    const form = useAppForm({
        onSubmit: async ({ value }) => {
            const filteredKeys = ['name', 'type'];
            const flags = Object.entries(value)
                .filter((v) => !filteredKeys.includes(v[0]))
                .reduce((acc, value) => {
                    if (value[1]) {
                        acc += value[0];
                    }
                    return acc;
                }, '');

            mutation.mutate({ name: value.name, type: value.type, flags });
        },
        defaultValues: {
            type: override?.type ?? 'command',
            name: override?.name ?? '',
            z: hasSMFlag('z', override),
            a: hasSMFlag('a', override),
            b: hasSMFlag('b', override),
            c: hasSMFlag('c', override),
            d: hasSMFlag('d', override),
            e: hasSMFlag('e', override),
            f: hasSMFlag('f', override),
            g: hasSMFlag('g', override),
            h: hasSMFlag('h', override),
            i: hasSMFlag('i', override),
            j: hasSMFlag('j', override),
            k: hasSMFlag('k', override),
            l: hasSMFlag('l', override),
            m: hasSMFlag('m', override),
            n: hasSMFlag('n', override),
            o: hasSMFlag('o', override),
            p: hasSMFlag('p', override),
            q: hasSMFlag('q', override),
            r: hasSMFlag('r', override),
            s: hasSMFlag('s', override),
            t: hasSMFlag('t', override)
        },
        validators: {
            onSubmit: z.object({
                type: z.enum(['command', 'group']),
                name: z.string().min(2),
                z: z.boolean(),
                a: z.boolean(),
                b: z.boolean(),
                c: z.boolean(),
                d: z.boolean(),
                e: z.boolean(),
                f: z.boolean(),
                g: z.boolean(),
                h: z.boolean(),
                i: z.boolean(),
                j: z.boolean(),
                k: z.boolean(),
                l: z.boolean(),
                m: z.boolean(),
                n: z.boolean(),
                o: z.boolean(),
                p: z.boolean(),
                q: z.boolean(),
                r: z.boolean(),
                s: z.boolean(),
                t: z.boolean()
            })
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
                    {override ? 'Edit' : 'Create'} Override
                </DialogTitle>

                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'name'}
                                children={(field) => {
                                    return <field.TextField label={'Name'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'type'}
                                children={(field) => {
                                    return (
                                        <field.SelectField
                                            label={'Override Type'}
                                            items={['command', 'group']}
                                            renderItem={(i) => {
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
                                    <form.CloseButton
                                        onClick={async () => {
                                            await modal.hide();
                                        }}
                                    />
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
