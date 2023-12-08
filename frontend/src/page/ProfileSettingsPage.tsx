import React, { JSX, useCallback } from 'react';
import ConstructionIcon from '@mui/icons-material/Construction';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { Accordion, AccordionDetails, AccordionSummary } from '@mui/material';
import Typography from '@mui/material/Typography';
import { Formik } from 'formik';
import { ContainerWithHeader } from '../component/ContainerWithHeader';

interface SettingsValues {
    signature: string;
}

export const ProfileSettingsPage = (): JSX.Element => {
    const [expanded, setExpanded] = React.useState<string | false>(false);

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
                initialValues={{ signature: '' }}
                onSubmit={onSubmit}
            >
                <Accordion
                    expanded={expanded === 'panel1'}
                    onChange={handleChange('panel1')}
                >
                    <AccordionSummary
                        expandIcon={<ExpandMoreIcon />}
                        aria-controls="panel1bh-content"
                        id="panel1bh-header"
                    >
                        <Typography sx={{ width: '33%', flexShrink: 0 }}>
                            General settings
                        </Typography>
                        <Typography sx={{ color: 'text.secondary' }}>
                            I am an accordion
                        </Typography>
                    </AccordionSummary>
                    <AccordionDetails>
                        <Typography>
                            Nulla facilisi. Phasellus sollicitudin nulla et quam
                            mattis feugiat. Aliquam eget maximus est, id
                            dignissim quam.
                        </Typography>
                    </AccordionDetails>
                </Accordion>
                <Accordion
                    expanded={expanded === 'panel2'}
                    onChange={handleChange('panel2')}
                >
                    <AccordionSummary
                        expandIcon={<ExpandMoreIcon />}
                        aria-controls="panel2bh-content"
                        id="panel2bh-header"
                    >
                        <Typography sx={{ width: '33%', flexShrink: 0 }}>
                            Users
                        </Typography>
                        <Typography sx={{ color: 'text.secondary' }}>
                            You are currently not an owner
                        </Typography>
                    </AccordionSummary>
                    <AccordionDetails>
                        <Typography>
                            Donec placerat, lectus sed mattis semper, neque
                            lectus feugiat lectus, varius pulvinar diam eros in
                            elit. Pellentesque convallis laoreet laoreet.
                        </Typography>
                    </AccordionDetails>
                </Accordion>
            </Formik>
        </ContainerWithHeader>
    );
};
