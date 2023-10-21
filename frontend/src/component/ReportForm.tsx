import EditNotificationsIcon from '@mui/icons-material/EditNotifications';
import Box from '@mui/material/Box';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import React, { JSX, useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
    apiCreateReport,
    BanReason,
    BanReasons,
    PlayerProfile,
    sessionKeyDemoName,
    sessionKeyReportPersonMessageIdName,
    sessionKeyReportSteamID
} from '../api';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { logErr } from '../util/errors';
import { ContainerWithHeader } from './ContainerWithHeader';
import { MDEditor } from './MDEditor';
import { PlayerMessageContext } from './PlayerMessageContext';
import { ProfileSelectionInput } from './ProfileSelectionInput';

export const ReportForm = (): JSX.Element => {
    const [reason, setReason] = useState<BanReason>(
        sessionStorage.getItem(sessionKeyReportPersonMessageIdName)
            ? BanReason.Language
            : BanReason.Cheating
    );
    const [body, setBody] = useState<string>('');
    const [reasonText, setReasonText] = useState<string>('');
    const [demoTick, setDemoTick] = useState(0);
    const [profile, setProfile] = useState<PlayerProfile | null>();
    const [inputSteamID, setInputSteamID] = useState<string>(
        sessionStorage.getItem(sessionKeyReportSteamID) ?? ''
    );
    const [personMessageID] = useState(
        parseInt(
            sessionStorage.getItem(sessionKeyReportPersonMessageIdName) ?? '0',
            10
        )
    );
    const [demoName] = useState(
        sessionStorage.getItem(sessionKeyDemoName) ?? ''
    );
    const { sendFlash } = useUserFlashCtx();
    const navigate = useNavigate();

    useEffect(() => {
        return () => {
            sessionStorage.removeItem(sessionKeyDemoName);
            sessionStorage.removeItem(sessionKeyReportPersonMessageIdName);
            sessionStorage.removeItem(sessionKeyReportSteamID);
        };
    }, []);

    const onSave = useCallback(
        (body_md: string) => {
            setBody(body_md);
            if (!profile || !profile.player.steam_id) {
                sendFlash('error', 'Invalid steam profile');
                return;
            }
            if (profile && body_md) {
                if (reason == BanReason.Custom && reasonText == '') {
                    sendFlash('error', 'Custom reason cannot be empty');
                    return;
                }
                apiCreateReport({
                    target_id: profile?.player.steam_id.toString(),
                    description: body_md,
                    reason: reason,
                    reason_text: reasonText,
                    demo_name: demoName,
                    demo_tick: demoTick,
                    person_message_id: personMessageID
                })
                    .then((response) => {
                        navigate(`/report/${response.report_id}`);
                    })
                    .catch((e) => {
                        sendFlash('error', `Failed to create report`);
                        logErr(e);
                    });
            }
        },
        [
            demoName,
            demoTick,
            navigate,
            personMessageID,
            profile,
            reason,
            reasonText,
            sendFlash
        ]
    );

    return (
        <ContainerWithHeader
            title={'Create a New Report'}
            iconLeft={<EditNotificationsIcon />}
            spacing={2}
        >
            <Box paddingLeft={2} paddingRight={2} paddingTop={3} width={'100%'}>
                <ProfileSelectionInput
                    fullWidth
                    input={inputSteamID}
                    setInput={setInputSteamID}
                    onProfileSuccess={(profile1) => {
                        setProfile(profile1);
                    }}
                />
                <FormControl margin={'normal'} variant={'filled'} fullWidth>
                    <InputLabel id="select_ban_reason_label">
                        Report Reason
                    </InputLabel>
                    <Select<BanReason>
                        labelId="select_ban_reason_label"
                        id="select_ban_reason"
                        value={reason}
                        fullWidth
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
                            BanReason.Language,
                            BanReason.Profile,
                            BanReason.ItemDescriptions,
                            BanReason.BotHost
                        ].map((v) => {
                            return (
                                <MenuItem value={v} key={v}>
                                    {BanReasons[v]}
                                </MenuItem>
                            );
                        })}
                    </Select>
                </FormControl>
                {reason == BanReason.Custom && (
                    <FormControl fullWidth>
                        <TextField
                            label={'Custom Reason'}
                            value={reasonText}
                            fullWidth
                            onChange={(event) => {
                                setReasonText(event.target.value);
                            }}
                        />
                    </FormControl>
                )}
                {demoName != '' && (
                    <Stack direction={'row'} spacing={2}>
                        <FormControl fullWidth>
                            <TextField
                                label={'Demo Name'}
                                value={demoName}
                                disabled={true}
                                fullWidth
                            />
                        </FormControl>
                        <FormControl fullWidth>
                            <TextField
                                label={'Demo Tick'}
                                value={demoTick}
                                fullWidth
                                onChange={(event) => {
                                    setDemoTick(parseInt(event.target.value));
                                }}
                            />
                        </FormControl>
                    </Stack>
                )}
                {personMessageID > 0 && (
                    <PlayerMessageContext
                        playerMessageId={personMessageID}
                        padding={5}
                    />
                )}
            </Box>
            <MDEditor
                initialBodyMDValue={body}
                onSave={onSave}
                saveLabel={'Create Report'}
            />
        </ContainerWithHeader>
    );
};
