import React, { ChangeEvent, useCallback, useEffect } from 'react';
import { getDistance } from '../util/gis';
import { useMapStateCtx } from '../contexts/MapStateCtx';
import {
    FormControlLabel,
    InputLabel,
    Select,
    Slider,
    Switch
} from '@mui/material';
import Grid from '@mui/material/Grid';
import Typography from '@mui/material/Typography';
import FormControl from '@mui/material/FormControl';
import MenuItem from '@mui/material/MenuItem';
import { uniq } from 'lodash-es';
import { SelectChangeEvent } from '@mui/material/Select';

// const useStyles = makeStyles((theme) => ({
//     root: {
//         display: 'flex',
//         padding: theme.spacing(2),
//         marginBottom: theme.spacing(2)
//     },
//     item: {
//         paddingBottom: 0
//     },
//     h1: { textAlign: 'left', fontSize: 20, marginBottom: 0 },
//     h2: { textAlign: 'center', fontSize: 16, marginBottom: 0 },
//     formControl: {
//         margin: theme.spacing(1),
//         minWidth: 120
//     }
// }));

export const ServerFilters = () => {
    const {
        setCustomRange,
        servers,
        customRange,
        pos,
        setSelectedServers,
        setFilterByRegion,
        setServers,
        filterByRegion,
        selectedRegion,
        setSelectedRegion,
        setShowOpenOnly,
        showOpenOnly
    } = useMapStateCtx();

    const regions = uniq([
        'any',
        ...(servers || []).map((value) => value.region)
    ]);

    const onRegionsChange = (event: SelectChangeEvent) => {
        setSelectedRegion(event.target.value);
    };

    const onShowOpenOnlyChanged = (
        _: ChangeEvent<HTMLInputElement>,
        checked: boolean
    ) => {
        setShowOpenOnly(checked);
    };

    const onRegionsToggleEnabledChanged = (
        _: ChangeEvent<HTMLInputElement>,
        checked: boolean
    ) => {
        setFilterByRegion(checked);
    };
    useEffect(() => {
        const defaultState = {
            showOpenOnly: false,
            selectedRegion: 'any',
            filterByRegion: false,
            customRange: 1500
        };
        let state = defaultState;
        try {
            const val = localStorage.getItem('filters');
            if (val) {
                state = JSON.parse(val);
            }
        } catch (e) {
            console.log(`Tried to load invalid filter state`);
            return;
        }
        setShowOpenOnly(state?.showOpenOnly || defaultState.showOpenOnly);
        setSelectedRegion(
            state?.selectedRegion != ''
                ? state.selectedRegion
                : defaultState.selectedRegion
        );
        setFilterByRegion(state?.filterByRegion || defaultState.filterByRegion);
        setCustomRange(state?.customRange || defaultState.customRange);
    }, [setCustomRange, setFilterByRegion, setSelectedRegion, setShowOpenOnly]);

    const saveFilterState = useCallback(() => {
        localStorage.setItem(
            'filters',
            JSON.stringify({
                showOpenOnly: showOpenOnly,
                selectedRegion: selectedRegion,
                filterByRegion: filterByRegion,
                customRange: customRange
            })
        );
    }, [customRange, filterByRegion, selectedRegion, showOpenOnly]);

    useEffect(() => {
        let s = servers;
        if (!filterByRegion && !selectedRegion.includes('any')) {
            s = s.filter((srv) => selectedRegion.includes(srv.region));
        }
        if (showOpenOnly) {
            s = s.filter(
                (srv) => (srv?.players?.length || 0) < (srv?.max_players || 32)
            );
        }
        if (filterByRegion && customRange) {
            s = s.filter(
                (srv) => getDistance(pos, srv.location) < customRange * 1000
            );
        }
        setSelectedServers(s);
        saveFilterState();
    }, [
        selectedRegion,
        showOpenOnly,
        filterByRegion,
        customRange,
        setServers,
        servers,
        setSelectedServers,
        saveFilterState,
        pos
    ]);

    const marks = [
        {
            value: 500,
            label: '500 km'
        },
        {
            value: 1500,
            label: '1500 km'
        },
        {
            value: 3000,
            label: '3000 km'
        },
        {
            value: 5000,
            label: '5000 km'
        }
    ];

    return (
        <Grid
            container
            style={{
                width: '100%',
                flexWrap: 'nowrap',
                alignItems: 'center'
                // justifyContent: 'center'
            }}
        >
            <Grid item xs={2}>
                <Typography variant={'h4'} align={'center'}>
                    Filters
                </Typography>
            </Grid>
            <Grid item xs>
                <FormControlLabel
                    control={
                        <Switch
                            checked={showOpenOnly}
                            onChange={onShowOpenOnlyChanged}
                            name="checkedA"
                        />
                    }
                    label="Open Slots"
                />
            </Grid>
            <Grid item xs>
                <FormControl>
                    <InputLabel id="region-selector-label">Region</InputLabel>
                    <Select<string>
                        disabled={filterByRegion}
                        labelId="region-selector-label"
                        id="region-selector"
                        value={selectedRegion}
                        onChange={onRegionsChange}
                    >
                        {regions.map((r) => {
                            return (
                                <MenuItem key={`region-${r}`} value={r}>
                                    {r}
                                </MenuItem>
                            );
                        })}
                    </Select>
                </FormControl>
            </Grid>
            <Grid item xs>
                <FormControlLabel
                    control={
                        <Switch
                            checked={filterByRegion}
                            onChange={onRegionsToggleEnabledChanged}
                            name="regionsEnabled"
                        />
                    }
                    label="By Range"
                />
            </Grid>
            <Grid item xs style={{ paddingRight: '2rem' }}>
                <Slider
                    style={{ zIndex: 1000 }}
                    disabled={!filterByRegion}
                    defaultValue={1000}
                    aria-labelledby="custom-range"
                    step={100}
                    max={5000}
                    valueLabelDisplay="auto"
                    value={customRange}
                    marks={marks}
                    onChange={(_: Event, value: number | number[]) => {
                        setCustomRange(value as number);
                    }}
                />
            </Grid>
        </Grid>
    );
};
