import React, { useCallback, useState } from 'react';
import Stack from '@mui/material/Stack';
import InputLabel from '@mui/material/InputLabel';
import Select from '@mui/material/Select';
import Box from '@mui/material/Box';
import FormControl from '@mui/material/FormControl';
import MenuItem from '@mui/material/MenuItem';
import {
    apiCreateReport,
    BanReason,
    BanReasons,
    PlayerProfile,
    SteamID
} from '../api';
import Typography from '@mui/material/Typography';
import { ProfileSelectionInput } from './ProfileSelectionInput';
import { logErr } from '../util/errors';
import { useNavigate } from 'react-router-dom';
import { MDEditor } from './MDEditor';

export const ReportForm = (): JSX.Element => {
    const [reason, setReason] = useState<BanReason>(BanReason.Cheating);
    const [body, setBody] = useState<string>('');
    const [profile, setProfile] = useState<PlayerProfile | null>();
    const [steamID, setSteamID] = useState<SteamID>(BigInt(0));
    const navigate = useNavigate();

    const onSave = useCallback(
        (body_md: string) => {
            setBody(body_md);
            if (profile && body_md) {
                apiCreateReport({
                    steam_id: profile?.player.steam_id,
                    description: body_md
                })
                    .then((report) => {
                        navigate(`/report/${report.report_id}`);
                    })
                    .catch(logErr);
            }
        },
        [navigate, profile]
    );

    return (
        <Stack spacing={3} padding={3}>
            <Box>
                <Typography variant={'h5'}>Create a New Report</Typography>
            </Box>
            <ProfileSelectionInput
                fullWidth
                input={steamID}
                setInput={setSteamID}
                onProfileSuccess={(profile1) => {
                    setProfile(profile1);
                }}
            />
            <FormControl margin={'normal'} variant={'filled'}>
                <InputLabel id="select_ban_reason_label">
                    Report Reason
                </InputLabel>
                <Select
                    labelId="select_ban_reason_label"
                    id="select_ban_reason"
                    value={reason}
                    variant={'outlined'}
                    label={'Ban Reason'}
                    onChange={(v) => {
                        setReason(v.target.value as BanReason);
                    }}
                >
                    {[
                        BanReason.Custom,
                        BanReason.External,
                        BanReason.Cheating,
                        BanReason.Racism,
                        BanReason.Harassment,
                        BanReason.Exploiting,
                        BanReason.WarningsExceeded,
                        BanReason.Spam,
                        BanReason.Language
                    ].map((v) => {
                        return (
                            <MenuItem value={v} key={v}>
                                {BanReasons[v]}
                            </MenuItem>
                        );
                    })}
                </Select>
            </FormControl>
            <MDEditor
                initialBodyMDValue={body}
                onSave={onSave}
                saveLabel={'Create Report'}
            />
        </Stack>
    );
};
