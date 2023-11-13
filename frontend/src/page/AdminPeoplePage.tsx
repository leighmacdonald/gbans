import React, { useEffect, useState, JSX, useCallback } from 'react';
import FilterListIcon from '@mui/icons-material/FilterList';
import PersonIcon from '@mui/icons-material/Person';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { fromUnixTime } from 'date-fns';
import { Formik } from 'formik';
import * as yup from 'yup';
import {
    apiSearchPeople,
    communityVisibilityState,
    defaultAvatarHash,
    permissionLevelString,
    Person
} from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { LazyTable, Order, RowsPerPage } from '../component/LazyTable';
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
    const [ip, setIP] = useState('');

    useEffect(() => {
        const abortController = new AbortController();
        setLoading(true);
        apiSearchPeople(
            {
                personaname: personaname,
                deleted: false,
                desc: sortOrder == 'desc',
                offset: page,
                limit: rowPerPageCount,
                order_by: sortColumn,
                steam_id: steamId,
                ip: ip
            },
            abortController
        )
            .then((response) => {
                setPeople(response.data);
                setTotalRows(response.count);
            })
            .catch((reason) => {
                logErr(reason);
            })
            .finally(() => setLoading(false));

        return () => abortController.abort('Cancelled');
    }, [
        ip,
        page,
        personaname,
        rowPerPageCount,
        sortColumn,
        sortOrder,
        steamId
    ]);

    const onFilterSubmit = useCallback((values: PeopleFilterValues) => {
        setSteamId(values.steam_id);
        setPersonaname(values.personaname);
        setIP(values.ip);
    }, []);

    const onFilterReset = useCallback(() => {
        setSteamId('');
        setPersonaname('');
        setIP('');
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
                        validateOnChange={true}
                        validateOnBlur={true}
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
                                tooltip: 'Is the player community banned',
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
                            }
                        ]}
                    />
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
};
