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
import {render} from "react-dom";
import * as React from "react";
import BanBrowser from "./component/BanBrowser";
import {AppealForm} from "./component/AppealForm";
import {PlayerBanForm} from "./component/PlayerBanForm";

// @ts-ignore
globalThis.jQuery = $

function main() {

    // @ts-ignore
    $(document).foundation();

    const p = window.location.pathname.toLowerCase()
    switch (p.toLowerCase()) {
        case "/":
            render(<BanBrowser/>, document.getElementById("ban_list"));
            break;
        case "/ban":
            render(<PlayerBanForm/>, document.getElementById("player_ban_form"));
            break;
        case "/appeal":
            render(<AppealForm  ban_id={0}/>, document.getElementById("appeal_form"))
    }
}

document.addEventListener("DOMContentLoaded", main)