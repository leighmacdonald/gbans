import * as React from "react";

interface props {
    ban_id: number
}

interface state {
    appeal_text: string
    appeal_state: number
    email: string

}

export class AppealForm extends React.Component<props, state> {
    componentDidMount() {
        // http(vars(Route.APIAppeal, {"ban_id": this.props.ban_id}),
        //     "GET",
        //     "",
        //     (resp) => {
        //         console.log(resp);
        //     }, (error: any) => {
        //         console.log(error);
        //     })
    }

    render() {
        return (
            <>
                <form>
                    <div className="grid-container">
                        <div className="grid-x grid-padding-x">
                            <div className="medium-12 cell">
                                <label>Input Label
                                    <input type="text" placeholder=".medium-6.cell"/>
                                </label>
                            </div>
                            <div className="medium-12 cell">
                                <label>Input Label
                                    <input type="text" placeholder=".medium-6.cell"/>
                                </label>
                            </div>
                        </div>
                    </div>
                </form>
            </>
        )
    }
}

