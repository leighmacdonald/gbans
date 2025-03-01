import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import GroupsIcon from '@mui/icons-material/Groups';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Link from '@mui/material/Link';
import MenuItem from '@mui/material/MenuItem';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import 'video-react/dist/video-react.css';
import { z } from 'zod';
import { apiCreateSMAdmin, apiSaveSMAdmin, AuthType, hasSMFlag, SMAdmin, SMGroups } from '../../api';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { numberStringValidator } from '../../util/validator/numberStringValidator.ts';
import { Heading } from '../Heading';
import { Buttons } from '../field/Buttons.tsx';
import { CheckboxSimple } from '../field/CheckboxSimple.tsx';
import { SelectFieldSimple } from '../field/SelectFieldSimple.tsx';
import { TextFieldSimple } from '../field/TextFieldSimple.tsx';

type mutateAdminArgs = {
    name: string;
    immunity: number;
    flags: string;
    auth_type: AuthType;
    identity: string;
    password: string;
};

export const SMAdminEditorModal = NiceModal.create(({ admin }: { admin?: SMAdmin; groups: SMGroups[] }) => {
    const modal = useModal();
    const { sendError } = useUserFlashCtx();

    const edit = useMutation({
        mutationKey: ['adminSMAdmin'],
        mutationFn: async ({ name, immunity, flags, auth_type, identity, password }: mutateAdminArgs) => {
            return admin?.admin_id
                ? await apiSaveSMAdmin(admin.admin_id, name, immunity, flags, auth_type, identity, password)
                : await apiCreateSMAdmin(name, immunity, flags, auth_type, identity, password);
        },
        onSuccess: async (admin) => {
            modal.resolve(admin);
            await modal.hide();
        },
        onError: sendError
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            const filteredKeys = ['name', 'immunity', 'auth_type', 'password', 'identity'];
            const flags = Object.entries(value)
                .filter((v) => !filteredKeys.includes(v[0]))
                .reduce((acc, value) => {
                    if (value[1]) {
                        acc += value[0];
                    }
                    return acc;
                }, '');
            edit.mutate({ ...value, immunity: Number(value.immunity), flags: flags });
        },
        validators: {
            onChange: z.object({
                auth_type: z.enum(['steam', 'name', 'ip']),
                identity: z.string().min(1),
                password: z.string(),
                immunity: z.string().transform(numberStringValidator(0, 100)),
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
        },
        defaultValues: {
            auth_type: admin?.auth_type ? admin.auth_type : 'steam',
            identity: admin?.identity ? admin.identity : '',
            password: admin?.password ? admin.password : '',
            name: admin?.name ?? '',
            immunity: admin?.immunity ? String(admin.immunity) : '',
            z: hasSMFlag('z', admin),
            a: hasSMFlag('a', admin),
            b: hasSMFlag('b', admin),
            c: hasSMFlag('c', admin),
            d: hasSMFlag('d', admin),
            e: hasSMFlag('e', admin),
            f: hasSMFlag('f', admin),
            g: hasSMFlag('g', admin),
            h: hasSMFlag('h', admin),
            i: hasSMFlag('i', admin),
            j: hasSMFlag('j', admin),
            k: hasSMFlag('k', admin),
            l: hasSMFlag('l', admin),
            m: hasSMFlag('m', admin),
            n: hasSMFlag('n', admin),
            o: hasSMFlag('o', admin),
            p: hasSMFlag('p', admin),
            q: hasSMFlag('q', admin),
            r: hasSMFlag('r', admin),
            s: hasSMFlag('s', admin),
            t: hasSMFlag('t', admin)
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
                    SM Admin Editor
                </DialogTitle>

                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid xs={6}>
                            <Field
                                name={'name'}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Alias'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'password'}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Password'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'auth_type'}
                                children={(props) => {
                                    return (
                                        <SelectFieldSimple
                                            {...props}
                                            label={'Auth Type'}
                                            fullwidth={true}
                                            items={['steam', 'name', 'ip']}
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
                                name={'identity'}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Identity'} fullwidth={true} />;
                                }}
                            />
                        </Grid>

                        <Grid xs={12}>
                            <Field
                                name={'immunity'}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Immunity Level'} fullwidth={true} />;
                                }}
                            />
                        </Grid>

                        <Grid xs={6}>
                            <Field
                                name={'z'}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(z) Full Admin'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'a'}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(a) Reserved Slot'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'b'}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(b) Generic Admin'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'c'}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(c) Kick Players'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'d'}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(d) Ban Players'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'e'}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(e) Unban Players'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'f'}
                                children={(props) => {
                                    return (
                                        <CheckboxSimple {...props} label={'(f) Slay/Harm Players'} fullwidth={true} />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'g'}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(g) Change Maps'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'h'}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(h) Change CVARs'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'i'}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(i) Exec Configs'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'j'}
                                children={(props) => {
                                    return (
                                        <CheckboxSimple
                                            {...props}
                                            label={'(j) Special Chat Privileges'}
                                            fullwidth={true}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'k'}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(k) Start Votes'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'l'}
                                children={(props) => {
                                    return (
                                        <CheckboxSimple {...props} label={'(l) Set Server Password'} fullwidth={true} />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'m'}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(m) RCON Access'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'n'}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(n) Enabled Cheats'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'o'}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(o) Custom Flag'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'p'}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(p) Custom Flag'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'q'}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(q) Custom Flag'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'r'}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(r) Custom Flag'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'s'}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(s) Custom Flag'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'t'}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(t) Custom Flag'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={12}>
                            <Link target={'_blank'} href={'https://wiki.alliedmods.net/Adding_Admins_(SourceMod)'}>
                                Additional Sourcemod Admin Info
                            </Link>
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
