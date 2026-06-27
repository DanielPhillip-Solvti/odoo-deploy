/** @odoo-module **/

import {Component} from "@odoo/owl";

export class LogsTab extends Component {
    static template = "deploy.LogsTab";
    static props = {
        env: {type: Object},
    };
}
