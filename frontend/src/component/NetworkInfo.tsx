import { useState } from 'react';
import SearchIcon from '@mui/icons-material/Search';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { Formik } from 'formik';
import { useNetworkQuery } from '../hooks/useNetworkQuery.ts';
import { LoadingPlaceholder } from './LoadingPlaceholder.tsx';
import { IPField, IPFieldProps } from './formik/IPField.tsx';
import { SubmitButton } from './modal/Buttons.tsx';

export const NetworkInfo = () => {
    const [ip, setIP] = useState('');

    const { data, loading } = useNetworkQuery({ ip: ip });

    const onSubmit = (values: IPFieldProps) => {
        setIP(values.ip);
    };

    return (
        <Grid container spacing={2}>
            <Grid xs={12}>
                <Formik onSubmit={onSubmit} initialValues={{ ip: '' }}>
                    <Grid
                        container
                        direction="row"
                        alignItems="top"
                        justifyContent="center"
                        spacing={2}
                    >
                        <Grid xs>
                            <IPField />
                        </Grid>
                        <Grid xs={2}>
                            <SubmitButton
                                label={'Submit'}
                                fullWidth
                                disabled={loading}
                                startIcon={<SearchIcon />}
                            />
                        </Grid>
                    </Grid>
                </Formik>
            </Grid>
            <Grid xs={12}>
                {loading ? (
                    <LoadingPlaceholder />
                ) : (
                    <Typography variant={'body1'}>
                        {JSON.stringify(data)}
                    </Typography>
                )}
            </Grid>
        </Grid>
    );
};
