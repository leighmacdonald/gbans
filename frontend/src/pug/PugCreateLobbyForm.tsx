import * as yup from 'yup';
import { useFormik } from 'formik';
import { Dialog, DialogContent, DialogTitle } from '@mui/material';
import { Heading } from '../component/Heading';
import GavelIcon from '@mui/icons-material/Gavel';
import Stack from '@mui/material/Stack';
import { ModalButtons } from '../component/formik/ModalButtons';
import React, { useEffect, useState } from 'react';
import {
    GameType,
    GameTypeField,
    gameTypeValidator
} from '../component/formik/GameTypeField';
import {
    MapSelectionField,
    mapValidator
} from '../component/formik/MapSelectionField';
import {
    GameConfig,
    GameConfigField,
    gameConfigValidator
} from '../component/formik/GameConfigField';
import {
    DescriptionField,
    descriptionValidator
} from '../component/formik/DescriptionField';
import { usePugCtx } from './PugCtx';
import {
    DiscordRequiredField,
    discordRequiredValidator
} from '../component/formik/DiscordRequiredField';
import {
    ServerSelectionField,
    serverValidator
} from '../component/formik/ServerSelectionField';
import { wsMsgTypePugCreateLobbyRequest } from './pug';

const validationSchema = yup.object({
    map_name: mapValidator,
    game_type: gameTypeValidator,
    game_config: gameConfigValidator,
    description: descriptionValidator,
    discord_required: discordRequiredValidator,
    server_name: serverValidator
});

interface PugCreateLobbyFormProps {
    open: boolean;
    setOpen: (open: boolean) => void;
}

export const PugCreateLobbyForm = ({
    open,
    setOpen
}: PugCreateLobbyFormProps) => {
    const [inProgress, setInProgress] = useState(false);
    const defaultValues = {
        map_name: 'cp_process_final',
        game_type: GameType.sixes,
        game_config: GameConfig.rgl,
        description: '',
        discord_required: false,
        server_name: 'sea-1'
    };

    const { createLobby, lobby } = usePugCtx();
    const formik = useFormik<wsMsgTypePugCreateLobbyRequest>({
        initialValues: defaultValues,
        validateOnBlur: true,
        validateOnChange: false,
        onReset: () => {
            alert('reset!');
        },
        validationSchema: validationSchema,
        onSubmit: async (opts) => {
            setInProgress(true);
            createLobby(opts);
        }
    });

    useEffect(() => {
        if (lobby?.lobbyId) {
            setInProgress(false);
        }
    }, [lobby]);

    const formId = 'PugCreateLobbyForm';
    return (
        <form onSubmit={formik.handleSubmit} id={formId}>
            <Dialog
                fullWidth
                open={open}
                onClose={() => {
                    setOpen(false);
                }}
            >
                <DialogTitle component={Heading} iconLeft={<GavelIcon />}>
                    Create Lobby
                </DialogTitle>

                <DialogContent>
                    <Stack spacing={2}>
                        <GameTypeField formik={formik} />
                        <MapSelectionField formik={formik} />
                        <GameConfigField formik={formik} />
                        <DescriptionField formik={formik} />
                        <DiscordRequiredField formik={formik} />
                        <ServerSelectionField formik={formik} />
                    </Stack>
                </DialogContent>
                <ModalButtons
                    formId={formId}
                    setOpen={setOpen}
                    inProgress={inProgress}
                />
            </Dialog>
        </form>
    );
};
