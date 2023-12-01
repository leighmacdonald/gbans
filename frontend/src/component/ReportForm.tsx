import React, { JSX, useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import EditNotificationsIcon from '@mui/icons-material/EditNotifications';
import SendIcon from '@mui/icons-material/Send';
import ButtonGroup from '@mui/material/ButtonGroup';
import FormControl from '@mui/material/FormControl';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import { Formik } from 'formik';
import {
    apiCreateReport,
    BanReason,
    sessionKeyDemoName,
    sessionKeyReportPersonMessageIdName,
    sessionKeyReportSteamID
} from '../api';
import { logErr } from '../util/errors';
import { ContainerWithHeader } from './ContainerWithHeader';
import { MDEditor } from './MDEditor';
import { PlayerMessageContext } from './PlayerMessageContext';
import { ProfileSelectionField } from './ProfileSelectionField';
import { BanReasonField } from './formik/BanReasonField';
import { BanReasonTextField } from './formik/BanReasonTextField';
import { ResetButton, SubmitButton } from './modal/Buttons';

interface ReportValues {
    steam_id: string;
    description: string;
    reason: BanReason;
    reason_text: string;
    demo_name: string;
    demo_tick: number;
    person_message_id: number;
}

export const ReportForm = (): JSX.Element => {
    const [demoTick, setDemoTick] = useState(0);

    const [personMessageID] = useState(
        parseInt(
            sessionStorage.getItem(sessionKeyReportPersonMessageIdName) ?? '0',
            10
        )
    );
    const [demoName] = useState(
        sessionStorage.getItem(sessionKeyDemoName) ?? ''
    );

    const navigate = useNavigate();

    useEffect(() => {
        return () => {
            sessionStorage.removeItem(sessionKeyDemoName);
            sessionStorage.removeItem(sessionKeyReportPersonMessageIdName);
            sessionStorage.removeItem(sessionKeyReportSteamID);
        };
    }, []);

    const onSubmit = useCallback(
        async (values: ReportValues) => {
            try {
                const report = await apiCreateReport({
                    demo_name: values.demo_name,
                    demo_tick: values.demo_tick,
                    description: values.description,
                    reason_text: values.reason_text,
                    target_id: values.steam_id,
                    person_message_id: values.person_message_id,
                    reason: values.reason
                });
                navigate(`/report/${report.report_id}`);
            } catch (e) {
                logErr(e);
            }
        },
        [navigate]
    );

    return (
        <ContainerWithHeader
            title={'Create a New Report'}
            iconLeft={<EditNotificationsIcon />}
            spacing={2}
        >
            <Formik<ReportValues>
                onSubmit={onSubmit}
                initialValues={{
                    demo_name: '',
                    demo_tick: 0,
                    person_message_id: 0,
                    description: '',
                    reason: BanReason.Cheating,
                    reason_text: '',
                    steam_id: ''
                }}
            >
                <Stack spacing={1}>
                    <ProfileSelectionField />
                    <BanReasonField />
                    <BanReasonTextField />

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
                                        setDemoTick(
                                            parseInt(event.target.value)
                                        );
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
                    <MDEditor />
                    <ButtonGroup>
                        <ResetButton />
                        <SubmitButton
                            label={'Report'}
                            startIcon={<SendIcon />}
                        />
                    </ButtonGroup>
                </Stack>
            </Formik>
        </ContainerWithHeader>
    );
};
