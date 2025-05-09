import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import RouterIcon from '@mui/icons-material/Router';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Grid from '@mui/material/Grid';
import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';
import { apiCreateServer, apiSaveServer, SaveServerOpts, Server } from '../../api';
import { useAppForm } from '../../contexts/formContext.tsx';
import { numberStringValidator } from '../../util/validator/numberStringValidator.ts';
import { Heading } from '../Heading';

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

    const form = useAppForm({
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
                    await form.handleSubmit();
                }}
            >
                <DialogTitle component={Heading} iconLeft={<RouterIcon />}>
                    Server {server?.server_id ? 'Editor' : 'Creator'}
                </DialogTitle>

                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 4 }}>
                            <form.AppField
                                name={'short_name'}
                                validators={{
                                    onChange: z.string().min(1)
                                }}
                                children={(field) => {
                                    return <field.TextField label={'Short Name/Tag'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 4 }}>
                            <form.AppField
                                name={'name'}
                                validators={{
                                    onChange: z.string().min(1)
                                }}
                                children={(field) => {
                                    return <field.TextField label={'Long Name'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 4 }}>
                            <form.AppField
                                name={'address'}
                                validators={{
                                    onChange: z.string().min(1)
                                }}
                                children={(field) => {
                                    return <field.TextField label={'Address'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 4 }}>
                            <form.AppField
                                name={'port'}
                                validators={{
                                    onChange: z.string().transform(numberStringValidator(1024, 65535))
                                }}
                                children={(field) => {
                                    return <field.TextField label={'Port'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 8 }}>
                            <form.AppField
                                name={'address_internal'}
                                validators={{
                                    onChange: z.string()
                                }}
                                children={(field) => {
                                    return <field.TextField label={'Address Internal'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 4 }}>
                            <form.AppField
                                name={'password'}
                                validators={{
                                    onChange: z.string().length(20)
                                }}
                                children={(field) => {
                                    return <field.TextField label={'Server Auth Key'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 4 }}>
                            <form.AppField
                                name={'rcon'}
                                validators={{
                                    onChange: z.string().min(6)
                                }}
                                children={(field) => {
                                    return <field.TextField label={'RCON Password'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 4 }}>
                            <form.AppField
                                name={'log_secret'}
                                validators={{
                                    onChange: z.string().transform(numberStringValidator(100000000, 999999999))
                                }}
                                children={(field) => {
                                    return <field.TextField label={'Log Secret'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'region'}
                                validators={{
                                    onChange: z.string().min(1)
                                }}
                                children={(field) => {
                                    return <field.TextField label={'Region'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'cc'}
                                validators={{
                                    onChange: z.string().length(2)
                                }}
                                children={(field) => {
                                    return <field.TextField label={'Country Code'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'latitude'}
                                validators={{
                                    onChange: z.string().transform(numberStringValidator(-99, 99))
                                }}
                                children={(field) => {
                                    return <field.TextField label={'Latitude'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'longitude'}
                                validators={{
                                    onChange: z.string().transform(numberStringValidator(-180, 180))
                                }}
                                children={(field) => {
                                    return <field.TextField label={'Longitude'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 4 }}>
                            <form.AppField
                                name={'reserved_slots'}
                                validators={{
                                    onChange: z.string()
                                }}
                                children={(field) => {
                                    return <field.TextField label={'Reserved Slots'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 4 }}>
                            <form.AppField
                                name={'is_enabled'}
                                validators={{
                                    onChange: z.boolean()
                                }}
                                children={(field) => {
                                    return <field.CheckboxField label={'Is Enabled'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 4 }}>
                            <form.AppField
                                name={'enabled_stats'}
                                validators={{
                                    onChange: z.boolean()
                                }}
                                children={(field) => {
                                    return <field.CheckboxField label={'Stats Enabled'} />;
                                }}
                            />
                        </Grid>
                    </Grid>
                </DialogContent>
                <DialogActions>
                    <Grid container>
                        <Grid size={{ xs: 12 }}>
                            <form.AppForm>
                                <form.CloseButton
                                    onClick={async () => {
                                        await modal.hide();
                                    }}
                                />
                                <form.ResetButton />
                                <form.SubmitButton />
                            </form.AppForm>
                        </Grid>
                    </Grid>
                </DialogActions>
            </form>
        </Dialog>
    );
});
