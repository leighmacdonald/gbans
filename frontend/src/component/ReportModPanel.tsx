import { useCallback } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import AutoFixNormalIcon from '@mui/icons-material/AutoFixNormal';
import GavelIcon from '@mui/icons-material/Gavel';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod/v4';
import { apiGetReport, apiReportSetState } from '../api';
import { useAppForm } from '../contexts/formContext.tsx';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { ReportStatus, ReportStatusCollection, ReportStatusEnum, reportStatusString } from '../schema/report.ts';
import { ContainerWithHeader } from './ContainerWithHeader';
import { ErrorDetails } from './ErrorDetails.tsx';
import { LoadingPlaceholder } from './LoadingPlaceholder.tsx';
import { ModalBanSteam } from './modal';

const schema = z.object({
    report_status: ReportStatusEnum
});

export const ReportModPanel = ({ reportId }: { reportId: number }) => {
    const queryClient = useQueryClient();
    const { sendFlash, sendError } = useUserFlashCtx();

    const {
        data: report,
        isLoading,
        isError,
        error
    } = useQuery({
        queryKey: ['report', { reportId }],
        queryFn: async () => {
            return await apiGetReport(Number(reportId));
        }
    });

    const stateMutation = useMutation({
        mutationKey: ['reportState', { report_status: report?.report_status }],
        mutationFn: async (report_status: ReportStatusEnum) => {
            return await apiReportSetState(Number(reportId), report_status);
        },
        onSuccess: async (_, reportStatus) => {
            if (!report) {
                return;
            }
            sendFlash(
                'success',
                `State changed from ${reportStatusString(
                    report?.report_status ?? ReportStatus.Opened
                )} => ${reportStatusString(reportStatus)}`
            );
            report.report_status = reportStatus;
        },
        onError: sendError
    });

    const onBan = useCallback(async () => {
        if (!report) {
            return;
        }

        try {
            const banRecord = await NiceModal.show(ModalBanSteam, {
                reportId: report.report_id,
                steamId: report.subject.steam_id
            });
            queryClient.setQueryData(['ban', { targetId: report?.target_id }], banRecord);
            stateMutation.mutate(ReportStatus.ClosedWithAction);
        } catch (e) {
            sendFlash('error', `Failed to ban: ${e}`);
        }
    }, [queryClient, report, sendFlash]);

    const form = useAppForm({
        onSubmit: async ({ value }) => {
            if (value.report_status == report?.report_status) {
                return;
            }
            stateMutation.mutate(value.report_status);
        },
        validators: { onSubmit: schema },
        defaultValues: { report_status: report?.report_status ?? ReportStatus.Opened }
    });

    if (isLoading) {
        return <LoadingPlaceholder />;
    }

    if (isError) {
        return <ErrorDetails error={error} />;
    }

    return (
        <form
            onSubmit={async (e) => {
                e.preventDefault();
                e.stopPropagation();
                await form.handleSubmit();
            }}
        >
            <ContainerWithHeader title={'Resolve Report'} iconLeft={<AutoFixNormalIcon />}>
                <List>
                    <ListItem>
                        <Stack sx={{ width: '100%' }} spacing={2}>
                            <form.AppField
                                name={'report_status'}
                                children={(field) => {
                                    return (
                                        <field.SelectField
                                            label={'Report State'}
                                            items={ReportStatusCollection}
                                            renderItem={(i) => {
                                                return (
                                                    <MenuItem key={i} value={i}>
                                                        {reportStatusString(i)}
                                                    </MenuItem>
                                                );
                                            }}
                                        />
                                    );
                                }}
                            />
                            <form.AppForm>
                                <ButtonGroup fullWidth>
                                    {report && (
                                        <Button
                                            variant={'contained'}
                                            color={'error'}
                                            startIcon={<GavelIcon />}
                                            onClick={onBan}
                                        >
                                            Ban Player
                                        </Button>
                                    )}
                                    <form.SubmitButton label={'Set State'} />
                                </ButtonGroup>
                            </form.AppForm>
                        </Stack>
                    </ListItem>
                </List>
            </ContainerWithHeader>
        </form>
    );
};
