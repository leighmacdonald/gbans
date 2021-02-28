import './scss/foundation.scss';
import './scss/app.scss';
import 'foundation-sites/dist/js/plugins/foundation.core';
import 'foundation-sites/dist/js/plugins/foundation.smoothScroll';
import 'foundation-sites/dist/js/plugins/foundation.equalizer';
import 'foundation-sites/dist/js/plugins/foundation.dropdownMenu';
import 'foundation-sites/dist/js/plugins/foundation.util.keyboard';
import 'foundation-sites/dist/js/plugins/foundation.util.mediaQuery';
import 'foundation-sites/dist/js/plugins/foundation.util.triggers';
import 'foundation-sites/dist/js/plugins/foundation.responsiveMenu';
import 'foundation-sites/dist/js/plugins/foundation.responsiveToggle';
import $ from 'jquery'
import 'what-input'
import * as React from "react";
import ReactDOM from 'react-dom';
import {App} from "./App"

// @ts-ignore
globalThis.jQuery = $

ReactDOM.render(<App/>, document.getElementById("root"))