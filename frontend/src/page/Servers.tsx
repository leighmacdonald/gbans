import React from "react";

export const Servers = () => {
    // @ts-ignore
    const [servers, setServers] = React.useState<any[]>([]);

    return (
        <div className="grid-container">
            <div className="grid-y grid-padding-y">
                <div className="cell medium-12 ">
                    <h1 className="text-center">Server Browser</h1>
                </div>
                <div className="cell large-12 medium-12" id="server_list">
                    <div className="grid-y grid-padding-y">
                        <div className="cell">
                            <div className="grid-x grid-padding-x heading">
                                <div className="cell medium-4"><i className="fi-list"/> Server Name</div>
                                <div className="cell medium-2">
                                    <i className="fi-info"/> Details
                                </div>
                                <div className="cell medium-3"><i className="fi-compass"/> Current Map</div>
                                <div className="cell medium-1"><i className="fi-torsos"/></div>
                                <div className="cell medium-2"><i className="fi-link"/> Connect</div>
                            </div>
                        </div>
                        {servers &&
                        <div className="cell row">
                            <div className="grid-x grid-padding-x">
                                <div className="cell medium-4">{"{{.ServerName}}"}</div>
                                <div className="cell medium-2">
                                    <div className="grid-x">
                                        <div className="cell large-4 text-center">
                                            <div className="game-{{ .A2SInfo.ID }}" />
                                        </div>
                                        <div className="cell large-4 text-center">
                                            <div className="vac-{{ .VacStatus }}" />
                                        </div>
                                        <div className="cell large-4 text-center">
                                            <div className="os-{{ .OS }}" />
                                        </div>
                                    </div>
                                </div>

                                <div className="cell medium-3 text-center">{"{{.Map}}"}</div>
                                <div className="cell medium-1 text-center">{"{{.PlayersCount}} / {{.Slots}}"}</div>
                                <div className="cell medium-2">
                                    <a className=" connect" href="steam://connect/{{ .Addr }}:{{ .Port }}">
                                        <i className="fi-target" /> Connect</a>
                                </div>
                            </div>
                        </div>
                        }
                    </div>
                </div>
            </div>
        </div>
    )
}