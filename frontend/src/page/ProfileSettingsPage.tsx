import React, { JSX, ReactNode, useCallback } from 'react';
import ConstructionIcon from '@mui/icons-material/Construction';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import Box from '@mui/material/Box';
import ButtonGroup from '@mui/material/ButtonGroup';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { Formik } from 'formik';
import {
    Accordion,
    AccordionDetails,
    AccordionSummary
} from '../component/Accordian';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { MDBodyField } from '../component/MDBodyField';
import { ResetButton, SubmitButton } from '../component/modal/Buttons';

interface SettingsValues {
    body_md: string;
}

const SettingRow = ({
    title,
    children
}: {
    title: string;
    children: ReactNode;
}) => {
    return (
        <>
            <Grid xs={2}>
                <Typography sx={{ width: '15%', flexShrink: 0 }}>
                    {title}
                </Typography>
            </Grid>
            <Grid xs={10}>{children}</Grid>
        </>
    );
};
export const ProfileSettingsPage = (): JSX.Element => {
    const [expanded, setExpanded] = React.useState<string | false>('general');

    const handleChange =
        (panel: string) => (_: React.SyntheticEvent, isExpanded: boolean) => {
            setExpanded(isExpanded ? panel : false);
        };

    const onSubmit = useCallback(async () => {}, []);

    return (
        <ContainerWithHeader
            title={'User Settings'}
            iconLeft={<ConstructionIcon />}
        >
            <Formik<SettingsValues>
                initialValues={{ body_md: '' }}
                onSubmit={onSubmit}
            >
                <>
                    <Accordion
                        expanded={expanded === 'general'}
                        onChange={handleChange('general')}
                    >
                        <AccordionSummary
                            expandIcon={<ExpandMoreIcon />}
                            aria-controls="general-content"
                            id="general-header"
                        >
                            <Typography sx={{ width: '16%', flexShrink: 0 }}>
                                General
                            </Typography>
                            <Typography sx={{ color: 'text.secondary' }}>
                                General account settings
                            </Typography>
                        </AccordionSummary>
                        <AccordionDetails>
                            <Typography>
                                Nulla facilisi. Phasellus sollicitudin nulla et
                                quam mattis feugiat. Aliquam eget maximus est,
                                id dignissim quam.
                            </Typography>
                        </AccordionDetails>
                    </Accordion>
                    <Accordion
                        expanded={expanded === 'forum'}
                        onChange={handleChange('forum')}
                    >
                        <AccordionSummary
                            expandIcon={<ExpandMoreIcon />}
                            aria-controls="forum-content"
                            id="forum-header"
                        >
                            <Typography sx={{ width: '16%', flexShrink: 0 }}>
                                Forum
                            </Typography>
                            <Typography sx={{ color: 'text.secondary' }}>
                                Configure forum signature and notification
                            </Typography>
                        </AccordionSummary>
                        <AccordionDetails>
                            <Grid container>
                                <SettingRow title={'Signature'}>
                                    <MDBodyField />
                                </SettingRow>
                            </Grid>
                        </AccordionDetails>
                    </Accordion>

                    <Box>
                        <ButtonGroup>
                            <ResetButton />
                            <SubmitButton label={'Save Settings'} />
                        </ButtonGroup>
                    </Box>
                </>
            </Formik>
        </ContainerWithHeader>
    );
};
