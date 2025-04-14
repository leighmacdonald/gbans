import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import GroupsIcon from '@mui/icons-material/Groups';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Grid from '@mui/material/Grid';
import Link from '@mui/material/Link';
import MenuItem from '@mui/material/MenuItem';
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
                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'name'}
                                validators={{
                                    onChange: z.string().min(2)
                                }}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Alias'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'password'}
                                validators={{
                                    onChange: z.string()
                                }}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Password'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'auth_type'}
                                validators={{
                                    onChange: z.enum(['steam', 'name', 'ip'])
                                }}
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
                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'identity'}
                                validators={{
                                    // TODO use proper validation based on selected auth type
                                    // Name requires password
                                    onChange: z.string().min(1)
                                }}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Identity'} fullwidth={true} />;
                                }}
                            />
                        </Grid>

                        <Grid size={{ xs: 12 }}>
                            <Field
                                name={'immunity'}
                                validators={{
                                    onChange: z.string().transform(numberStringValidator(0, 100))
                                }}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Immunity Level'} fullwidth={true} />;
                                }}
                            />
                        </Grid>

                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'z'}
                                validators={{ onChange: z.boolean() }}
                                children={({ state, handleBlur, handleChange }) => {
                                    return (
                                        <CheckboxSimple
                                            state={state}
                                            handleBlur={handleBlur}
                                            handleChange={handleChange}
                                            label={'(z) Full Admin'}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'a'}
                                validators={{ onChange: z.boolean() }}
                                children={({ state, handleBlur, handleChange }) => {
                                    return (
                                        <CheckboxSimple
                                            state={state}
                                            handleBlur={handleBlur}
                                            handleChange={handleChange}
                                            label={'(a) Reserved Slot'}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'b'}
                                validators={{ onChange: z.boolean() }}
                                children={({ state, handleBlur, handleChange }) => {
                                    return (
                                        <CheckboxSimple
                                            state={state}
                                            handleBlur={handleBlur}
                                            handleChange={handleChange}
                                            label={'(b) Generic Admin'}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'c'}
                                validators={{ onChange: z.boolean() }}
                                children={({ state, handleBlur, handleChange }) => {
                                    return (
                                        <CheckboxSimple
                                            state={state}
                                            handleBlur={handleBlur}
                                            handleChange={handleChange}
                                            label={'(c) Kick Players'}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'d'}
                                validators={{ onChange: z.boolean() }}
                                children={({ state, handleBlur, handleChange }) => {
                                    return (
                                        <CheckboxSimple
                                            state={state}
                                            handleBlur={handleBlur}
                                            handleChange={handleChange}
                                            label={'(d) Ban Players'}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'e'}
                                validators={{ onChange: z.boolean() }}
                                children={({ state, handleBlur, handleChange }) => {
                                    return (
                                        <CheckboxSimple
                                            state={state}
                                            handleBlur={handleBlur}
                                            handleChange={handleChange}
                                            label={'(e) Unban Players'}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'f'}
                                validators={{ onChange: z.boolean() }}
                                children={({ state, handleBlur, handleChange }) => {
                                    return (
                                        <CheckboxSimple
                                            state={state}
                                            handleBlur={handleBlur}
                                            handleChange={handleChange}
                                            label={'(f) Slay/Harm Players'}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'g'}
                                validators={{ onChange: z.boolean() }}
                                children={({ state, handleBlur, handleChange }) => {
                                    return (
                                        <CheckboxSimple
                                            state={state}
                                            handleBlur={handleBlur}
                                            handleChange={handleChange}
                                            label={'(g) Change Maps'}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'h'}
                                validators={{ onChange: z.boolean() }}
                                children={({ state, handleBlur, handleChange }) => {
                                    return (
                                        <CheckboxSimple
                                            state={state}
                                            handleBlur={handleBlur}
                                            handleChange={handleChange}
                                            label={'(h) Change CVARs'}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'i'}
                                validators={{ onChange: z.boolean() }}
                                children={({ state, handleBlur, handleChange }) => {
                                    return (
                                        <CheckboxSimple
                                            state={state}
                                            handleBlur={handleBlur}
                                            handleChange={handleChange}
                                            label={'(i) Exec Configs'}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'j'}
                                validators={{ onChange: z.boolean() }}
                                children={({ state, handleBlur, handleChange }) => {
                                    return (
                                        <CheckboxSimple
                                            state={state}
                                            handleBlur={handleBlur}
                                            handleChange={handleChange}
                                            label={'(j) Special Chat Privileges'}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'k'}
                                validators={{ onChange: z.boolean() }}
                                children={({ state, handleBlur, handleChange }) => {
                                    return (
                                        <CheckboxSimple
                                            state={state}
                                            handleBlur={handleBlur}
                                            handleChange={handleChange}
                                            label={'(k) Start Votes'}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'l'}
                                validators={{ onChange: z.boolean() }}
                                children={({ state, handleBlur, handleChange }) => {
                                    return (
                                        <CheckboxSimple
                                            state={state}
                                            handleBlur={handleBlur}
                                            handleChange={handleChange}
                                            label={'(l) Set Server Password'}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'m'}
                                validators={{ onChange: z.boolean() }}
                                children={({ state, handleBlur, handleChange }) => {
                                    return (
                                        <CheckboxSimple
                                            state={state}
                                            handleBlur={handleBlur}
                                            handleChange={handleChange}
                                            label={'(m) RCON Access'}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'n'}
                                validators={{ onChange: z.boolean() }}
                                children={({ state, handleBlur, handleChange }) => {
                                    return (
                                        <CheckboxSimple
                                            state={state}
                                            handleBlur={handleBlur}
                                            handleChange={handleChange}
                                            label={'(n) Enabled Cheats'}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'o'}
                                validators={{ onChange: z.boolean() }}
                                children={({ state, handleBlur, handleChange }) => {
                                    return (
                                        <CheckboxSimple
                                            state={state}
                                            handleBlur={handleBlur}
                                            handleChange={handleChange}
                                            label={'(o) Custom Flag'}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'p'}
                                validators={{ onChange: z.boolean() }}
                                children={({ state, handleBlur, handleChange }) => {
                                    return (
                                        <CheckboxSimple
                                            state={state}
                                            handleBlur={handleBlur}
                                            handleChange={handleChange}
                                            label={'(p) Custom Flag'}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'q'}
                                validators={{ onChange: z.boolean() }}
                                children={({ state, handleBlur, handleChange }) => {
                                    return (
                                        <CheckboxSimple
                                            state={state}
                                            handleBlur={handleBlur}
                                            handleChange={handleChange}
                                            label={'(q) Custom Flag'}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'r'}
                                validators={{ onChange: z.boolean() }}
                                children={({ state, handleBlur, handleChange }) => {
                                    return (
                                        <CheckboxSimple
                                            state={state}
                                            handleBlur={handleBlur}
                                            handleChange={handleChange}
                                            label={'(r) Custom Flag'}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'s'}
                                validators={{ onChange: z.boolean() }}
                                children={({ state, handleBlur, handleChange }) => {
                                    return (
                                        <CheckboxSimple
                                            state={state}
                                            handleBlur={handleBlur}
                                            handleChange={handleChange}
                                            label={'(s) Custom Flag'}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'t'}
                                validators={{ onChange: z.boolean() }}
                                children={({ state, handleBlur, handleChange }) => {
                                    return (
                                        <CheckboxSimple
                                            state={state}
                                            handleBlur={handleBlur}
                                            handleChange={handleChange}
                                            label={'(t) Custom Flag'}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 12 }}>
                            <Link target={'_blank'} href={'https://wiki.alliedmods.net/Adding_Admins_(SourceMod)'}>
                                Additional Sourcemod Admin Info
                            </Link>
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
