import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import RouterIcon from '@mui/icons-material/Router';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Grid from '@mui/material/Grid2';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';
import { apiCreateServer, apiSaveServer, SaveServerOpts, Server } from '../../api';
import { numberStringValidator } from '../../util/validator/numberStringValidator.ts';
import { Heading } from '../Heading';
import { Buttons } from '../field/Buttons.tsx';
import { CheckboxSimple } from '../field/CheckboxSimple.tsx';
import { TextFieldSimple } from '../field/TextFieldSimple.tsx';

type ServerEditValues = {
    short_name: string;
    name: string;
    address: string;
    address_internal: string;
    port: string;
    password: string;
    rcon: string;
    region: string;
    cc: string;
    latitude: string;
    longitude: string;
    reserved_slots: string;
    is_enabled: boolean;
    enabled_stats: boolean;
    log_secret: string;
};

export const ServerEditorModal = NiceModal.create(({ server }: { server?: Server }) => {
    const modal = useModal();

    const mutation = useMutation({
        mutationKey: ['adminServer'],
        mutationFn: async (values: ServerEditValues) => {
            const opts: SaveServerOpts = {
                server_name_short: values.short_name,
                server_name: values.name,
                host: values.address,
                port: Number(values.port),
                password: values.password,
                rcon: values.rcon,
                region: values.region,
                cc: values.cc,
                lat: Number(values.latitude),
                lon: Number(values.longitude),
                reserved_slots: Number(values.reserved_slots),
                is_enabled: values.is_enabled,
                enable_stats: values.enabled_stats,
                log_secret: Number(values.log_secret),
                address_internal: values.address_internal
            };
            if (server?.server_id) {
                modal.resolve(await apiSaveServer(server.server_id, opts));
            } else {
                modal.resolve(await apiCreateServer(opts));
            }
            await modal.hide();
        }
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutation.mutate(value);
        },
        defaultValues: {
            short_name: server ? server.short_name : '',
            name: server ? server.name : '',
            address: server ? server.address : '',
            port: server ? String(server.port) : '27015',
            password: server ? server.password : '',
            rcon: server ? server.rcon : '',
            region: server ? server.region : '',
            cc: server ? server.cc : '',
            latitude: server ? String(server.latitude) : '',
            longitude: server ? String(server.longitude) : '',
            reserved_slots: server ? String(server.reserved_slots) : '0',
            is_enabled: server ? server.is_enabled : true,
            enabled_stats: server ? server.enable_stats : true,
            log_secret: server ? String(server.log_secret) : '',
            address_internal: server ? server.address_internal : ''
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
                <DialogTitle component={Heading} iconLeft={<RouterIcon />}>
                    Server {server?.server_id ? 'Editor' : 'Creator'}
                </DialogTitle>

                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 4 }}>
                            <Field
                                name={'short_name'}
                                validators={{
                                    onChange: z.string().min(1)
                                }}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Short Name/Tag'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 4 }}>
                            <Field
                                name={'name'}
                                validators={{
                                    onChange: z.string().min(1)
                                }}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Long Name'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 4 }}>
                            <Field
                                name={'address'}
                                validators={{
                                    onChange: z.string().min(1)
                                }}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Address'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 4 }}>
                            <Field
                                name={'port'}
                                validators={{
                                    onChange: z.string().transform(numberStringValidator(1024, 65535))
                                }}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Port'} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={8}>
                            <Field
                                name={'address_internal'}
                                validators={{
                                    onChange: z.string()
                                }}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Address Internal'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 4 }}>
                            <Field
                                name={'password'}
                                validators={{
                                    onChange: z.string().length(20)
                                }}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Server Auth Key'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 4 }}>
                            <Field
                                name={'rcon'}
                                validators={{
                                    onChange: z.string().min(6)
                                }}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'RCON Password'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 4 }}>
                            <Field
                                name={'log_secret'}
                                validators={{
                                    onChange: z.string().transform(numberStringValidator(100000000, 999999999))
                                }}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Log Secret'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'region'}
                                validators={{
                                    onChange: z.string().min(1)
                                }}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Region'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'cc'}
                                validators={{
                                    onChange: z.string().length(2)
                                }}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Country Code'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'latitude'}
                                validators={{
                                    onChange: z.string().transform(numberStringValidator(-99, 99))
                                }}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Latitude'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <Field
                                name={'longitude'}
                                validators={{
                                    onChange: z.string().transform(numberStringValidator(-180, 180))
                                }}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Longitude'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 4 }}>
                            <Field
                                name={'reserved_slots'}
                                validators={{
                                    onChange: z.string()
                                }}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Reserved Slots'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 4 }}>
                            <Field
                                name={'is_enabled'}
                                validators={{
                                    onChange: z.boolean()
                                }}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'Is Enabled'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 4 }}>
                            <Field
                                name={'enabled_stats'}
                                validators={{
                                    onChange: z.boolean()
                                }}
                                children={(props) => {
                                    return <CheckboxSimple {...props} label={'Stats Enabled'} />;
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
