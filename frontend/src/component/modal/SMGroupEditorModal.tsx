import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import GroupsIcon from '@mui/icons-material/Groups';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Grid from '@mui/material/Grid';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import 'video-react/dist/video-react.css';
import { z } from 'zod';
import { apiCreateSMGroup, apiSaveSMGroup, hasSMFlag, SMGroups } from '../../api';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { numberStringValidator } from '../../util/validator/numberStringValidator.ts';
import { Heading } from '../Heading';
import { Buttons } from '../field/Buttons.tsx';
import { CheckboxSimple } from '../field/CheckboxSimple.tsx';
import { TextFieldSimple } from '../field/TextFieldSimple.tsx';

export const SMGroupEditorModal = NiceModal.create(({ group }: { group?: SMGroups }) => {
    const modal = useModal();
    const { sendError } = useUserFlashCtx();

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

    const { Field, Subscribe, handleSubmit, reset } = useForm({
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
        defaultValues: {
            name: group?.name ?? '',
            immunity: group?.immunity_level ? String(group.immunity_level) : '',
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
                    SM Group Editor
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
                                    return <TextFieldSimple {...props} label={'Group Name'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
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
