import { ReportStatus } from '../api';
import CancelPresentationIcon from '@mui/icons-material/CancelPresentation';
import GavelIcon from '@mui/icons-material/Gavel';
import QuizIcon from '@mui/icons-material/Quiz';
import NewReleasesIcon from '@mui/icons-material/NewReleases';
import React from 'react';
import { Tooltip } from '@mui/material';

interface ReportStatusIconProps {
    reportStatus: ReportStatus;
}

export const ReportStatusIcon = ({
    reportStatus
}: ReportStatusIconProps): JSX.Element => {
    switch (reportStatus) {
        case ReportStatus.NeedMoreInfo:
            return (
                <Tooltip title={'Needs more information'}>
                    <QuizIcon color={'warning'} />
                </Tooltip>
            );
        case ReportStatus.ClosedWithoutAction:
            return (
                <Tooltip title={'Report closed with no action'}>
                    <CancelPresentationIcon color={'action'} />
                </Tooltip>
            );
        case ReportStatus.ClosedWithAction:
            return (
                <Tooltip title={'Report closed with action'}>
                    <GavelIcon color={'error'} />
                </Tooltip>
            );
        case ReportStatus.Opened:
            return (
                <Tooltip title={'New report'}>
                    <NewReleasesIcon color={'success'} />
                </Tooltip>
            );
    }
};
