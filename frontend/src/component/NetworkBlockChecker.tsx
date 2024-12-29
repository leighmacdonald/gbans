import { useCallback, useState } from 'react';
import SendIcon from '@mui/icons-material/Send';
import Grid from '@mui/material/Grid2';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Formik } from 'formik';
import * as yup from 'yup';
import { apiCIDRBlockCheck, CIDRBlockCheckResponse } from '../api';
import { logErr } from '../util/errors';
import { ipFieldValidator } from '../util/validators';
import { VCenterBox } from './VCenterBox';
import { SubmitButton } from './modal/Buttons';

interface NetworkBlockCheckerValues {
    ip: string;
}

const validationSchema = yup.object({ ip: ipFieldValidator });

export const NetworkBlockChecker = () => {
    const [status, setStatus] = useState<CIDRBlockCheckResponse>();

    const onSubmit = useCallback(async (values: NetworkBlockCheckerValues) => {
        try {
            const resp = await apiCIDRBlockCheck(values.ip);
            setStatus(resp);
        } catch (e) {
            logErr(e);
            setStatus(undefined);
        }
    }, []);

    return (
        <Formik<NetworkBlockCheckerValues>
            initialValues={{ ip: '' }}
            onSubmit={onSubmit}
            validationSchema={validationSchema}
        >
            <Grid container spacing={1}>
                <Grid size={{ xs: 12 }}>
                    <Typography variant={'body2'} padding={1}>
                        Check if an IP is currently blocked via cidr ban sources.
                    </Typography>
                </Grid>
                <Grid size={{ xs: 8 }}>
                    <Stack>
                        <IPField />
                    </Stack>
                </Grid>
                <Grid size={{ xs: 4 }}>
                    <VCenterBox>
                        <SubmitButton label={'Check IP'} startIcon={<SendIcon />} />
                    </VCenterBox>
                </Grid>
                {status != undefined && (
                    <Grid size={{ xs: 12 }}>
                        {status.blocked ? (
                            <Typography variant={'body1'} color={'error'}>
                                Blocked: True Source: {status.source}
                            </Typography>
                        ) : (
                            <Typography variant={'body1'} color={'success'}>
                                Blocked: False
                            </Typography>
                        )}
                    </Grid>
                )}
            </Grid>
        </Formik>
    );
};
