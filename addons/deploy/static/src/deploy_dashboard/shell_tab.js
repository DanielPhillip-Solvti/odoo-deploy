/** @odoo-module **/

import {Component} from "@odoo/owl";

export class ShellTab extends Component {
    static template = "deploy.ShellTab";
    static props = {
        env: {type: Object},
        agent: {type: Object, optional: true},
    };
}
