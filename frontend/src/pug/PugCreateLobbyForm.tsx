import * as yup from 'yup';
import { useFormik } from 'formik';
import { logErr } from '../util/errors';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { Dialog, DialogContent, DialogTitle } from '@mui/material';
import { Heading } from '../component/Heading';
import GavelIcon from '@mui/icons-material/Gavel';
import Stack from '@mui/material/Stack';
import { ModalButtons } from '../component/formik/ModalButtons';
import React from 'react';
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

interface PugCreateLobbyFormValues {
    gameType: GameType;
    gameConfig: GameConfig;
    map: string;
    description: string;
    discord_required: boolean;
}

const validationSchema = yup.object({
    map: mapValidator,
    gameType: gameTypeValidator,
    gameConfig: gameConfigValidator,
    description: descriptionValidator
});

interface PugCreateLobbyFormProps {
    open: boolean;
    setOpen: (open: boolean) => void;
}

export const PugCreateLobbyForm = ({
    open,
    setOpen
}: PugCreateLobbyFormProps) => {
    const defaultValues = {
        map: 'cp_process_final',
        gameType: GameType.sixes,
        gameConfig: GameConfig.rgl,
        description: '',
        discord_required: false
    };
    const { sendFlash } = useUserFlashCtx();
    const formik = useFormik<PugCreateLobbyFormValues>({
        initialValues: defaultValues,
        validateOnBlur: true,
        validateOnChange: false,
        onReset: () => {
            alert('reset!');
        },
        validationSchema: validationSchema,
        onSubmit: async (_) => {
            try {
                sendFlash('success', 'Lobby created successfully');
            } catch (e) {
                logErr(e);
                sendFlash('error', 'Error saving ban');
            } finally {
                setOpen(false);
            }
        }
    });

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
                    </Stack>
                </DialogContent>
                <ModalButtons formId={formId} setOpen={setOpen} />
            </Dialog>
        </form>
    );
};
