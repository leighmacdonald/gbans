import CancelPresentationIcon from '@mui/icons-material/CancelPresentation';
import GavelIcon from '@mui/icons-material/Gavel';
import NewReleasesIcon from '@mui/icons-material/NewReleases';
import QuizIcon from '@mui/icons-material/Quiz';
import Tooltip from '@mui/material/Tooltip';
import { ReportStatus } from '../api';

interface ReportStatusIconProps {
    reportStatus: ReportStatus;
}

export const ReportStatusIcon = ({ reportStatus }: ReportStatusIconProps): JSX.Element => {
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
        default:
            return (
                <Tooltip title={'New report'}>
                    <NewReleasesIcon color={'success'} />
                </Tooltip>
            );
    }
};
