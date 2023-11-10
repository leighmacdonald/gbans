import React, { ChangeEvent, useCallback, useEffect, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormGroup from '@mui/material/FormGroup';
import Stack from '@mui/material/Stack';
import Switch from '@mui/material/Switch';
import TextField from '@mui/material/TextField';
import {
    apiCreateServer,
    SaveServerOpts,
    Server,
    apiSaveServer
} from '../../api';
import { useUserFlashCtx } from '../../contexts/UserFlashCtx';
import { logErr } from '../../util/errors';
import { Nullable } from '../../util/types';
import { Heading } from '../Heading';
import { ConfirmationModal, ConfirmationModalProps } from './ConfirmationModal';

export interface ServerEditorModalProps extends ConfirmationModalProps<Server> {
    server: Nullable<Server>;
}

export const ServerEditorModal = NiceModal.create(
    ({ onSuccess, server }: ServerEditorModalProps) => {
        const [serverId, setServerId] = useState<number>(0);
        const [serverName, setServerName] = useState<string>(
            server?.short_name ?? ''
        );
        const [serverNameLong, setServerNameLong] = useState<string>(
            server?.name ?? ''
        );
        const [address, setAddress] = useState<string>('');
        const [port, setPort] = useState<number>(7015);
        const [password, setPassword] = useState<string>('');
        const [rcon, setRcon] = useState<string>('');
        const [region, setRegion] = useState<string>('');
        const [countryCode, setCountryCode] = useState<string>('');
        const [latitude, setLatitude] = useState<number>(0.0);
        const [longitude, setLongitude] = useState<number>(0.0);
        const [reservedSlots, setReservedSlots] = useState<number>(0);
        const [playersMax, setPlayersMax] = useState<number>(24);
        const [isEnabled, setIsEnabled] = useState<boolean>(false);

        useEffect(() => {
            setServerId(server?.server_id ?? 0);
            setServerName(server?.short_name ?? '');
            setServerNameLong(server?.name ?? '');
            setAddress(server?.address ?? '');
            setPort(server?.port ?? 27015);
            setPassword(server?.password ?? '');
            setRcon(server?.rcon ?? '');
            setRegion(server?.region ?? '');
            setCountryCode(server?.cc ?? '');

            setReservedSlots(server?.reserved_slots ?? 0);
            setPlayersMax(server?.players_max ?? 24);
            if (server) {
                setLatitude(server?.latitude);
                setLongitude(server?.longitude);
                setIsEnabled(server?.is_enabled);
            }
        }, [server]);

        const { sendFlash } = useUserFlashCtx();

        const handleSubmit = useCallback(() => {
            if (
                !serverName ||
                !serverNameLong ||
                !address ||
                !rcon ||
                !countryCode ||
                port <= 0 ||
                port > 65535
            ) {
                sendFlash('error', 'Invalid values');
                return;
            }
            const opts: SaveServerOpts = {
                port: port,
                cc: countryCode,
                host: address,
                rcon: rcon,
                lat: latitude,
                lon: longitude,
                server_name: serverNameLong,
                server_name_short: serverName,
                region: region,
                reserved_slots: reservedSlots,
                is_enabled: isEnabled
            };
            if (serverId > 0) {
                apiSaveServer(serverId, opts)
                    .then((response) => {
                        sendFlash(
                            'success',
                            `Server saved successfully: ${response.short_name}`
                        );
                        onSuccess && onSuccess(response);
                    })
                    .catch((err) => {
                        sendFlash('error', `Failed to save server: ${err}`);
                    });
            } else {
                apiCreateServer(opts)
                    .then((response) => {
                        sendFlash(
                            'success',
                            `Server created successfully: ${response.short_name}`
                        );
                        onSuccess && onSuccess(response);
                    })
                    .catch((err) => {
                        sendFlash('error', `Failed to create server: ${err}`);
                    });
            }
        }, [
            serverName,
            serverNameLong,
            address,
            rcon,
            countryCode,
            port,
            longitude,
            latitude,
            region,
            reservedSlots,
            isEnabled,
            serverId,
            sendFlash,
            onSuccess
        ]);

        return (
            <ConfirmationModal
                id={'modal-server-editor'}
                onAccept={() => {
                    handleSubmit();
                }}
                aria-labelledby="modal-title"
                aria-describedby="modal-description"
            >
                <Stack spacing={2}>
                    <Heading>Server Editor</Heading>

                    <Stack spacing={3} alignItems={'center'}>
                        <TextField
                            fullWidth
                            id={'server_name'}
                            label={'Server Name'}
                            value={serverName}
                            onChange={(evt: ChangeEvent<HTMLInputElement>) => {
                                setServerName(evt.target.value);
                            }}
                        />

                        <TextField
                            fullWidth
                            value={serverNameLong}
                            id={'server_name_long'}
                            label={'Server Name Long'}
                            onChange={(evt: ChangeEvent<HTMLInputElement>) => {
                                setServerNameLong(evt.target.value);
                            }}
                        />

                        <FormGroup>
                            <FormControlLabel
                                checked={isEnabled}
                                control={<Switch />}
                                label="Enabled"
                                onChange={(_, enabled) => setIsEnabled(enabled)}
                            />
                        </FormGroup>

                        <TextField
                            fullWidth
                            id={'address'}
                            label={'Host Address'}
                            value={address}
                            onChange={(evt: ChangeEvent<HTMLInputElement>) => {
                                setAddress(evt.target.value);
                            }}
                        />

                        <TextField
                            fullWidth
                            id={'port'}
                            label={'Server Port'}
                            value={port}
                            onChange={(evt: ChangeEvent<HTMLInputElement>) => {
                                try {
                                    setPort(parseInt(evt.target.value, 10));
                                } catch (e) {
                                    logErr(e);
                                }
                            }}
                        />

                        <TextField
                            fullWidth
                            id={'password'}
                            label={'Server Password'}
                            value={password}
                            onChange={(evt: ChangeEvent<HTMLInputElement>) => {
                                setPassword(evt.target.value);
                            }}
                        />

                        <TextField
                            fullWidth
                            id={'rcon'}
                            label={'RCON Password'}
                            value={rcon}
                            onChange={(evt: ChangeEvent<HTMLInputElement>) => {
                                setRcon(evt.target.value);
                            }}
                        />

                        <TextField
                            fullWidth
                            id={'region'}
                            label={'Region'}
                            value={region}
                            onChange={(evt: ChangeEvent<HTMLInputElement>) => {
                                setRegion(evt.target.value);
                            }}
                        />

                        <TextField
                            fullWidth
                            id={'cc'}
                            label={'Country Code'}
                            value={countryCode}
                            onChange={(evt: ChangeEvent<HTMLInputElement>) => {
                                setCountryCode(evt.target.value);
                            }}
                        />
                        <Stack direction={'row'}>
                            <TextField
                                fullWidth
                                id={'latitude'}
                                label={'Latitude'}
                                value={latitude}
                                onChange={(
                                    evt: ChangeEvent<HTMLInputElement>
                                ) => {
                                    try {
                                        setLatitude(
                                            parseFloat(evt.target.value)
                                        );
                                    } catch (e) {
                                        logErr(e);
                                    }
                                }}
                            />
                            <TextField
                                fullWidth
                                id={'longitude'}
                                label={'Longitude'}
                                value={longitude}
                                onChange={(
                                    evt: ChangeEvent<HTMLInputElement>
                                ) => {
                                    try {
                                        setLongitude(
                                            parseFloat(evt.target.value)
                                        );
                                    } catch (e) {
                                        logErr(e);
                                    }
                                }}
                            />
                        </Stack>

                        <TextField
                            fullWidth
                            id={'reserved_slots'}
                            label={'Reserved Slots'}
                            value={reservedSlots}
                            onChange={(evt: ChangeEvent<HTMLInputElement>) => {
                                try {
                                    setReservedSlots(
                                        parseInt(evt.target.value, 10)
                                    );
                                } catch (e) {
                                    logErr(e);
                                }
                            }}
                        />

                        <TextField
                            fullWidth
                            id={'players_max'}
                            label={'Players Max'}
                            value={playersMax}
                            onChange={(evt: ChangeEvent<HTMLInputElement>) => {
                                try {
                                    setPlayersMax(
                                        parseInt(evt.target.value, 10)
                                    );
                                } catch (e) {
                                    logErr(e);
                                }
                            }}
                        />
                    </Stack>
                </Stack>
            </ConfirmationModal>
        );
    }
);
