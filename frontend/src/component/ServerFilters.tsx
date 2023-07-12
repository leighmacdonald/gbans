import React, { ChangeEvent, useCallback, useEffect } from 'react';
import { SelectChangeEvent } from '@mui/material/Select';
import FormControlLabel from '@mui/material/FormControlLabel';
import InputLabel from '@mui/material/InputLabel';
import Switch from '@mui/material/Switch';
import Paper from '@mui/material/Paper';
import Select from '@mui/material/Select';
import Slider from '@mui/material/Slider';
import Grid from '@mui/material/Unstable_Grid2';
import FormControl from '@mui/material/FormControl';
import MenuItem from '@mui/material/MenuItem';
import styled from '@mui/system/styled';
import { uniq } from 'lodash-es';
import { useMapStateCtx } from '../contexts/MapStateCtx';
import { Heading } from './Heading';

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
        let s = servers.sort((a, b) => {
            // Sort by position if we have a non-default position.
            // otherwise, sort by server name
            if (pos.lat !== 0) {
                if (a.distance > b.distance) {
                    return 1;
                }
                if (a.distance < b.distance) {
                    return -1;
                }
                return 0;
            }
            return ('' + a.name_short).localeCompare(b.name_short);
        });
        if (!filterByRegion && !selectedRegion.includes('any')) {
            s = s.filter((srv) => selectedRegion.includes(srv.region));
        }
        if (showOpenOnly) {
            s = s.filter(
                (srv) => (srv?.players || 0) < (srv?.max_players || 32)
            );
        }
        if (filterByRegion && customRange && customRange > 0) {
            s = s.filter((srv) => srv.distance < customRange);
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
        <Paper elevation={1}>
            <Grid
                container
                spacing={2}
                style={{
                    width: '100%',
                    flexWrap: 'nowrap',
                    alignItems: 'center',
                    padding: 10
                    // justifyContent: 'center'
                }}
            >
                <Grid xs={2}>
                    <Heading>Filters</Heading>
                </Grid>
                <Grid xs>
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
                <Grid xs>
                    <FormControl>
                        <InputLabel id="region-selector-label">
                            Region
                        </InputLabel>
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
                <Grid xs>
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
                <Grid xs style={{ paddingRight: '2rem' }}>
                    <RangeSlider
                        style={{
                            zIndex: 1000
                        }}
                        disabled={!filterByRegion}
                        defaultValue={1000}
                        aria-labelledby="custom-range"
                        step={100}
                        max={5000}
                        valueLabelDisplay="off"
                        value={customRange}
                        marks={marks}
                        onChange={(_: Event, value: number | number[]) => {
                            setCustomRange(value as number);
                        }}
                    />
                </Grid>
            </Grid>
        </Paper>
    );
};

const RangeSlider = styled(Slider)(({ theme }) => ({
    height: 2,
    padding: '15px 0',
    '& .MuiSlider-thumb': {
        backgroundColor: theme.palette.common.white
    },
    '& .MuiSlider-valueLabel': {
        color: theme.palette.common.white,
        '&:before': {
            display: 'none'
        },
        '& *': {
            background: 'transparent',
            color: theme.palette.common.white
        }
    },
    '& .MuiSlider-track': {
        border: 'none'
    },
    '& .MuiSlider-rail': {
        opacity: 0.5,
        backgroundColor: '#bfbfbf'
    },
    '& .MuiSlider-mark': {
        backgroundColor: '#bfbfbf',
        height: 8,
        width: 1,
        '&.MuiSlider-markActive': {
            opacity: 1,
            backgroundColor: 'currentColor'
        }
    }
}));
