import React, { useEffect, useState, JSX, useCallback } from 'react';
import FilterListIcon from '@mui/icons-material/FilterList';
import PersonIcon from '@mui/icons-material/Person';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { fromUnixTime } from 'date-fns';
import format from 'date-fns/format';
import { Formik } from 'formik';
import * as yup from 'yup';
import {
    apiSearchPeople,
    communityVisibilityState,
    defaultAvatarHash,
    permissionLevelString,
    Person,
    PlayerQuery
} from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { Order, RowsPerPage } from '../component/DataTable';
import { LazyTable } from '../component/LazyTable';
import { LoadingSpinner } from '../component/LoadingSpinner';
import { PersonCell } from '../component/PersonCell';
import { nonResolvingSteamIDInputTest } from '../component/formik/AuthorIdField';
import { FilterButtons } from '../component/formik/FilterButtons';
import {
    PersonanameField,
    personanameFieldValidator
} from '../component/formik/PersonanameField';
import { SteamIdField } from '../component/formik/SteamIdField';
import { logErr } from '../util/errors';

export const steamIDValidatorSimple = yup
    .string()
    .label('Player Steam ID')
    .test('steam_id', 'Invalid steamid', nonResolvingSteamIDInputTest);

const validationSchema = yup.object({
    steam_id: steamIDValidatorSimple,
    personaname: personanameFieldValidator
});

interface PeopleFilterValues {
    steam_id: string;
    personaname: string;
}

export const AdminPeople = (): JSX.Element => {
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] = useState<keyof Person>('created_on');
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        RowsPerPage.TwentyFive
    );
    const [page, setPage] = useState(0);
    const [loading, setLoading] = useState(false);
    const [totalRows, setTotalRows] = useState<number>(0);
    const [people, setPeople] = useState<Person[]>([]);
    const [steamId, setSteamId] = useState('');
    const [personaname, setPersonaname] = useState('');

    useEffect(() => {
        const abortController = new AbortController();
        const opts: PlayerQuery = {
            personaname: personaname,
            deleted: false,
            desc: sortOrder == 'desc',
            offset: page,
            limit: rowPerPageCount,
            order_by: sortColumn,
            steam_id: steamId
        };
        setLoading(true);
        apiSearchPeople(opts, abortController)
            .then((response) => {
                setPeople(response.data);
                setTotalRows(response.count);
            })
            .catch((reason) => {
                logErr(reason);
            })
            .finally(() => setLoading(false));

        return () => abortController.abort('Cancelled');
    }, [page, personaname, rowPerPageCount, sortColumn, sortOrder, steamId]);

    const onFilterSubmit = useCallback((values: PeopleFilterValues) => {
        setSteamId(values.steam_id);
        setPersonaname(values.personaname);
    }, []);

    const onFilterReset = useCallback(() => {
        setSteamId('');
        setPersonaname('');
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
                        initialValues={{ personaname: '', steam_id: '' }}
                        validateOnChange={true}
                        validationSchema={validationSchema}
                    >
                        <Grid container>
                            <Grid xs={12} padding={2}>
                                <Stack direction={'row'} spacing={2}>
                                    <SteamIdField fullWidth />
                                    <PersonanameField />
                                </Stack>
                            </Grid>
                            <Grid xs={12} padding={2}>
                                <FilterButtons />
                            </Grid>
                        </Grid>
                    </Formik>
                </ContainerWithHeader>
            </Grid>
            <Grid xs={12}>
                <ContainerWithHeader
                    title={'Player Search'}
                    iconLeft={loading ? <LoadingSpinner /> : <PersonIcon />}
                >
                    <LazyTable
                        count={totalRows}
                        sortOrder={sortOrder}
                        sortColumn={sortColumn}
                        onSortColumnChanged={async (column) => {
                            setSortColumn(column);
                        }}
                        onSortOrderChanged={async (direction) => {
                            setSortOrder(direction);
                        }}
                        onRowsPerPageChange={(
                            event: React.ChangeEvent<
                                HTMLInputElement | HTMLTextAreaElement
                            >
                        ) => {
                            setRowPerPageCount(
                                parseInt(event.target.value, 10)
                            );
                            setPage(0);
                        }}
                        onPageChange={(_, newPage) => {
                            setPage(newPage);
                        }}
                        rows={people}
                        showPager
                        page={page}
                        rowsPerPage={rowPerPageCount}
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
                                tooltip: 'Amount of vac bans',
                                sortKey: 'vac_bans',
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
                                        {format(
                                            fromUnixTime(row.timecreated),
                                            'yyyy-MM-dd'
                                        )}
                                    </Typography>
                                )
                            },
                            {
                                label: 'Created',
                                tooltip: 'When the user was first seen',
                                sortable: true,
                                sortKey: 'created_on',
                                align: 'left',
                                width: '150px',
                                renderer: (obj) => {
                                    return (
                                        <Typography variant={'body1'}>
                                            {format(
                                                obj.created_on,
                                                'yyyy-MM-dd'
                                            )}
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
                            }
                        ]}
                    />
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
};
