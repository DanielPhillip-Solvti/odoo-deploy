/** @odoo-module **/

import {Component} from "@odoo/owl";

export class MonitorTab extends Component {
    static template = "deploy.MonitorTab";
    static props = {
        env: {type: Object},
        heartbeat: {type: Object, optional: true},
    };

    get heartbeatJSON() {
        return this.props.heartbeat ? JSON.stringify(this.props.heartbeat, null, 2) : "";
    }
}
