import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import GroupsIcon from '@mui/icons-material/Groups';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Grid from '@mui/material/Grid';
import MenuItem from '@mui/material/MenuItem';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import 'video-react/dist/video-react.css';
import { z } from 'zod';
import { apiCreateSMOverrides, apiSaveSMOverrides, hasSMFlag, OverrideType, SMOverrides } from '../../api';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { Heading } from '../Heading';
import { Buttons } from '../field/Buttons.tsx';
import { CheckboxSimple } from '../field/CheckboxSimple.tsx';
import { SelectFieldSimple } from '../field/SelectFieldSimple.tsx';
import { TextFieldSimple } from '../field/TextFieldSimple.tsx';

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

    const { Field, Subscribe, handleSubmit, reset } = useForm({
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
                    {override ? 'Edit' : 'Create'} Override
                </DialogTitle>

                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'name'}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Name'} fullwidth={true} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
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
});
