import * as React from "react";
import {EventHandler} from "react";

interface ISearchProps {
    onInputChange: (evt: any) => {}
}

interface ISearchBarState {
    query: string
}

export default class SearchBar extends React.Component<ISearchProps, ISearchBarState> {
    constructor(props) {
        super(props);
        this.onInputChange = this.onInputChange.bind(this);
    }

    onInputChange(evt) {
        this.setState({...this.state, query: evt.target.value})
        this.props.onInputChange(evt.target.value)
    }

    render() {
        return (
            <div className={"grid-x grid-padding-x"}>
                <div className={"cell medium-6 small-12 medium-offset-3"}>
                    <input type={"text"} placeholder={"Steamid, Name"} onChange={this.onInputChange} />
                </div>
            </div>
        );
    }
}