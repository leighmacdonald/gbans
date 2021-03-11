import * as React from "react";
import ReactDOM from 'react-dom';
import {App} from "./App"
import {CssBaseline} from "@material-ui/core";
import { ThemeProvider } from '@material-ui/core/styles';
import default_theme from "./themes/default_theme";

ReactDOM.render(
    <ThemeProvider theme={default_theme}>
        <CssBaseline />
        <App/>
    </ThemeProvider>,
    document.getElementById("root"))