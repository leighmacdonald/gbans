import React, { useState } from 'react';
import Box from '@mui/material/Box';
import { PlayerBanForm } from './PlayerBanForm';
import { ProfileSelectionInput } from './ProfileSelectionInput';
import Stack from '@mui/material/Stack';
import { Ban, SteamID } from '../api';
import { ConfirmationModal, ConfirmationModalProps } from './ConfirmationModal';

export interface BanModalProps extends ConfirmationModalProps {
    ban?: Ban;
}

export const BanModal = ({ open, setOpen }: BanModalProps) => {
    const [steamId, setSteamId] = useState<SteamID>(BigInt(0));
    const [input, setInput] = useState<SteamID>(BigInt(0));

    return (
        <ConfirmationModal
            open={open}
            setOpen={setOpen}
            aria-labelledby="modal-modal-title"
            aria-describedby="modal-modal-description"
        >
            <Box padding={2}>
                <Stack spacing={2}>
                    <ProfileSelectionInput
                        fullWidth
                        onProfileSuccess={(profile) => {
                            if (profile) {
                                setSteamId(profile.player.steam_id);
                            } else {
                                setSteamId(BigInt(0));
                            }
                        }}
                        input={input}
                        setInput={setInput}
                    />
                    <PlayerBanForm steamId={steamId} />
                </Stack>
            </Box>
        </ConfirmationModal>
    );
};
