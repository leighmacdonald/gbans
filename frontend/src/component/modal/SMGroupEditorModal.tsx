import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import GroupsIcon from '@mui/icons-material/Groups';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { zodValidator } from '@tanstack/zod-form-adapter';
import 'video-react/dist/video-react.css';
import { z } from 'zod';
import { apiCreateSMGroup, apiSaveSMGroup, hasFlag, SMGroups } from '../../api';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { numberStringValidator } from '../../util/validator/numberStringValidator.ts';
import { Heading } from '../Heading';
import { Buttons } from '../field/Buttons.tsx';
import { CheckboxSimple } from '../field/CheckboxSimple.tsx';
import { TextFieldSimple } from '../field/TextFieldSimple.tsx';

export const SMGroupEditorModal = NiceModal.create(({ group }: { group?: SMGroups }) => {
    const modal = useModal();
    const { sendFlash } = useUserFlashCtx();

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
        onError: (error) => {
            sendFlash('error', `Failed to create group: ${error}`);
        }
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
        validatorAdapter: zodValidator,
        defaultValues: {
            name: group?.name ?? '',
            immunity: group?.immunity_level ? String(group.immunity_level) : '',
            z: hasFlag('z', group),
            a: hasFlag('a', group),
            b: hasFlag('b', group),
            c: hasFlag('c', group),
            d: hasFlag('d', group),
            e: hasFlag('e', group),
            f: hasFlag('f', group),
            g: hasFlag('g', group),
            h: hasFlag('h', group),
            i: hasFlag('i', group),
            j: hasFlag('j', group),
            k: hasFlag('k', group),
            l: hasFlag('l', group),
            m: hasFlag('m', group),
            n: hasFlag('n', group),
            o: hasFlag('o', group),
            p: hasFlag('p', group),
            q: hasFlag('q', group),
            r: hasFlag('r', group),
            s: hasFlag('s', group),
            t: hasFlag('t', group)
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
                        <Grid xs={6}>
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
                        <Grid xs={6}>
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

                        <Grid xs={6}>
                            <Field
                                name={'z'}
                                validators={{ onChange: z.boolean() }}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(z) Full Admin'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'a'}
                                validators={{ onChange: z.boolean() }}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(a) Reserved Slot'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'b'}
                                validators={{ onChange: z.boolean() }}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(b) Generic Admin'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'c'}
                                validators={{ onChange: z.boolean() }}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(c) Kick Players'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'d'}
                                validators={{ onChange: z.boolean() }}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(d) Ban Players'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'e'}
                                validators={{ onChange: z.boolean() }}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(e) Unban Players'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'f'}
                                validators={{ onChange: z.boolean() }}
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
                                validators={{ onChange: z.boolean() }}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(g) Change Maps'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'h'}
                                validators={{ onChange: z.boolean() }}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(h) Change CVARs'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'i'}
                                validators={{ onChange: z.boolean() }}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(i) Exec Configs'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'j'}
                                validators={{ onChange: z.boolean() }}
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
                                validators={{ onChange: z.boolean() }}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(k) Start Votes'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'l'}
                                validators={{ onChange: z.boolean() }}
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
                                validators={{ onChange: z.boolean() }}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(m) RCON Access'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'n'}
                                validators={{ onChange: z.boolean() }}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(n) Enabled Cheats'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'o'}
                                validators={{ onChange: z.boolean() }}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(o) Custom Flag'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'p'}
                                validators={{ onChange: z.boolean() }}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(p) Custom Flag'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'q'}
                                validators={{ onChange: z.boolean() }}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(q) Custom Flag'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'r'}
                                validators={{ onChange: z.boolean() }}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(r) Custom Flag'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'s'}
                                validators={{ onChange: z.boolean() }}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(s) Custom Flag'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'t'}
                                validators={{ onChange: z.boolean() }}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'(t) Custom Flag'} fullwidth={true} />;
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
