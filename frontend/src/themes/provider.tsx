import {MuiThemeProvider} from '@material-ui/core';
import lightTheme from './light';
import darkTheme from './dark';
import React, {createContext, FC, useState} from 'react';
import {Theme} from '@material-ui/core/styles';

type ThemeName = 'lightTheme' | 'darkTheme';

const themes: Record<ThemeName, Theme> = {
    lightTheme: lightTheme,
    darkTheme: darkTheme
};

export const ThemeContext = createContext((_: ThemeName): void => {});

const ThemeProvider: FC = props => {
    const curThemeName = (localStorage.getItem('appTheme') || 'darkTheme') as ThemeName;
    const [themeName, _setThemeName] = useState<ThemeName>(curThemeName);
    const setThemeName = (themeName: ThemeName) => {
        localStorage.setItem('appTheme', themeName);
        _setThemeName(themeName);
    };
    const theme = themes[themeName];
    return (
        <ThemeContext.Provider value={setThemeName}>
            <MuiThemeProvider theme={theme}>{props.children}</MuiThemeProvider>
        </ThemeContext.Provider>
    );
};

export default ThemeProvider;
