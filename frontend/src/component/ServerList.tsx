import {apiGetServers, Server} from "../util/api";
import {CreateDataTable} from "./DataTable";

export const ServerList = () => {
    return CreateDataTable<Server>()({
        connector: async () => {
            return await apiGetServers() as Promise<Server[]>
        },
        id_field: "server_id",
        heading: "Servers",
        headers: [
            {id: "server_id",disablePadding: false, label: "Created", numeric: true},
            {id: "server_name",disablePadding: false, label: "Created", numeric: false},
            {id: "address",disablePadding: false, label: "Created", numeric: false},
            {id: "port",disablePadding: false, label: "Created", numeric: true},
            {id: "rcon",disablePadding: false, label: "Created", numeric: false},
            {id: "token_created_on",disablePadding: false, label: "Created", numeric: false}
        ]
    })
}