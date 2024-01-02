import React, { JSX, useCallback, useMemo } from 'react';
import useUrlState from '@ahooksjs/use-url-state';
import NiceModal from '@ebay/nice-modal-react';
import FilterListIcon from '@mui/icons-material/FilterList';
import PersonSearchIcon from '@mui/icons-material/PersonSearch';
import VpnKeyIcon from '@mui/icons-material/VpnKey';
import { IconButton } from '@mui/material';
import ButtonGroup from '@mui/material/ButtonGroup';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { fromUnixTime } from 'date-fns';
import { Formik } from 'formik';
import * as yup from 'yup';
import {
    communityVisibilityState,
    defaultAvatarHash,
    PermissionLevel,
    permissionLevelString,
    Person
} from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { LoadingSpinner } from '../component/LoadingSpinner';
import { PersonCell } from '../component/PersonCell';
import { FilterButtons } from '../component/formik/FilterButtons';
import { IPField, ipFieldValidator } from '../component/formik/IPField';
import {
    PersonanameField,
    personanameFieldValidator
} from '../component/formik/PersonanameField';
import { nonResolvingSteamIDInputTest } from '../component/formik/SourceIdField';
import { SteamIdField } from '../component/formik/SteamIdField';
import { ModalPersonEditor } from '../component/modal';
import { LazyTable, RowsPerPage } from '../component/table/LazyTable';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { usePeople } from '../hooks/usePeople';
import { logErr } from '../util/errors';
import { isValidSteamDate, renderDate } from '../util/text';

export const steamIDValidatorSimple = yup
    .string()
    .label('Player Steam ID')
    .test('steam_id', 'Invalid steamid', nonResolvingSteamIDInputTest);

const validationSchema = yup.object({
    steam_id: steamIDValidatorSimple,
    personaname: personanameFieldValidator,
    ip: ipFieldValidator
});

interface PeopleFilterValues {
    steam_id: string;
    personaname: string;
    ip: string;
}

export const AdminPeoplePage = (): JSX.Element => {
    const [state, setState] = useUrlState({
        page: undefined,
        steam_id: undefined,
        personaname: undefined,
        ip: undefined,
        rows: undefined,
        sortOrder: undefined,
        sortColumn: undefined
    });

    const { currentUser } = useCurrentUserCtx();

    const isAdmin = useMemo(() => {
        return currentUser.permission_level == PermissionLevel.Admin;
    }, [currentUser.permission_level]);

    const { data, count, loading } = usePeople({
        personaname: state.personaname,
        deleted: false,
        desc: state.sortOrder == 'desc',
        limit: Number(state.rows ?? RowsPerPage.Ten),
        offset: Number((state.page ?? 0) * (state.rows ?? RowsPerPage.Ten)),
        order_by: state.sortColumn ?? 'created_on',
        steam_id: state.steam_id,
        ip: state.ip
    });

    const onFilterSubmit = useCallback(
        (values: PeopleFilterValues) => {
            setState(values);
        },
        [setState]
    );

    const onFilterReset = useCallback(() => {
        setState({
            ip: '',
            personaname: '',
            steam_id: ''
        });
    }, [setState]);

    const onEditPerson = useCallback(async (person: Person) => {
        try {
            await NiceModal.show<Person>(ModalPersonEditor, {
                person
            });
        } catch (e) {
            logErr(e);
        }
    }, []);

    return (
        <Grid container spacing={2}>
            <Grid xs={12}>
                <ContainerWithHeader
                    title={'Person Filters'}
                    iconLeft={<FilterListIcon />}
                >
                    <Formik
                        onSubmit={onFilterSubmit}
                        onReset={onFilterReset}
                        initialValues={{
                            personaname: '',
                            steam_id: '',
                            ip: ''
                        }}
                        validationSchema={validationSchema}
                    >
                        <Grid container spacing={2}>
                            <Grid xs>
                                <SteamIdField />
                            </Grid>
                            <Grid xs>
                                <PersonanameField />
                            </Grid>
                            <Grid xs>
                                <IPField />
                            </Grid>
                            <Grid xs>
                                <FilterButtons />
                            </Grid>
                        </Grid>
                    </Formik>
                </ContainerWithHeader>
            </Grid>
            <Grid xs={12}>
                <ContainerWithHeader
                    title={'Player Search'}
                    iconLeft={
                        loading ? <LoadingSpinner /> : <PersonSearchIcon />
                    }
                >
                    <LazyTable
                        count={count}
                        sortOrder={state.sortOrder}
                        sortColumn={state.sortColumn}
                        onSortColumnChanged={async (column) => {
                            setState({ sortColumn: column });
                        }}
                        onSortOrderChanged={async (direction) => {
                            setState({ sortOrder: direction });
                        }}
                        onRowsPerPageChange={(
                            event: React.ChangeEvent<
                                HTMLInputElement | HTMLTextAreaElement
                            >
                        ) => {
                            setState({
                                rows: Number(event.target.value),
                                page: 0
                            });
                        }}
                        onPageChange={(_, newPage) => {
                            setState({ page: newPage });
                        }}
                        rows={data}
                        showPager
                        page={Number(state.page ?? 0)}
                        rowsPerPage={Number(
                            state.rows ?? RowsPerPage.TwentyFive
                        )}
                        columns={[
                            {
                                label: 'Steam ID',
                                tooltip: 'Steam ID',
                                sortKey: 'steam_id',
                                align: 'left',
                                sortable: true,
                                renderer: (row) => (
                                    <PersonCell
                                        steam_id={row.steam_id}
                                        personaname={
                                            row.personaname != ''
                                                ? row.personaname
                                                : row.steam_id
                                        }
                                        avatar_hash={
                                            row.avatarhash != ''
                                                ? row.avatarhash
                                                : defaultAvatarHash
                                        }
                                    />
                                )
                            },
                            {
                                label: 'Profile',
                                tooltip: 'Community Visibility State',
                                sortKey: 'communityvisibilitystate',
                                align: 'left',
                                sortable: true,
                                renderer: (row) => (
                                    <Typography variant={'body1'}>
                                        {row.communityvisibilitystate ==
                                        communityVisibilityState.Public
                                            ? 'Public'
                                            : 'Private'}
                                    </Typography>
                                )
                            },
                            {
                                label: 'Vac Ban',
                                tooltip: 'Amount of vac bans',
                                sortKey: 'vac_bans',
                                align: 'left',
                                sortable: true,
                                renderer: (row) => (
                                    <Typography variant={'body1'}>
                                        {row.vac_bans}
                                    </Typography>
                                )
                            },
                            {
                                label: 'Comm. Ban',
                                tooltip: 'Is the player community banned',
                                sortKey: 'community_banned',
                                align: 'left',
                                sortable: true,
                                renderer: (row) => (
                                    <Typography variant={'body1'}>
                                        {row.community_banned ? 'Yes' : 'No'}
                                    </Typography>
                                )
                            },
                            {
                                label: 'Account Created',
                                tooltip: 'When the account was created',
                                sortKey: 'timecreated',
                                align: 'left',
                                sortable: true,
                                renderer: (row) => (
                                    <Typography variant={'body1'}>
                                        {!isValidSteamDate(
                                            fromUnixTime(row.timecreated)
                                        )
                                            ? 'Unknown'
                                            : renderDate(
                                                  fromUnixTime(row.timecreated)
                                              )}
                                    </Typography>
                                )
                            },
                            {
                                label: 'First Seen',
                                tooltip: 'When the user was first seen',
                                sortable: true,
                                sortKey: 'created_on',
                                align: 'left',
                                width: '150px',
                                renderer: (obj) => {
                                    return (
                                        <Typography variant={'body1'}>
                                            {renderDate(obj.created_on)}
                                        </Typography>
                                    );
                                }
                            },
                            {
                                label: 'Perms',
                                tooltip: 'Permission Level',
                                sortKey: 'permission_level',
                                align: 'left',
                                sortable: true,
                                renderer: (row) => (
                                    <Typography variant={'body1'}>
                                        {permissionLevelString(
                                            row.permission_level
                                        )}
                                    </Typography>
                                )
                            },
                            {
                                virtual: true,
                                virtualKey: 'actions',
                                label: '',
                                tooltip: '',
                                align: 'right',
                                renderer: (obj) => {
                                    return (
                                        <ButtonGroup>
                                            <IconButton
                                                disabled={!isAdmin}
                                                color={'warning'}
                                                onClick={async () => {
                                                    try {
                                                        await onEditPerson(obj);
                                                    } catch (e) {
                                                        logErr(e);
                                                    }
                                                }}
                                            >
                                                <VpnKeyIcon />
                                            </IconButton>
                                        </ButtonGroup>
                                    );
                                }
                            }
                        ]}
                    />
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
};
