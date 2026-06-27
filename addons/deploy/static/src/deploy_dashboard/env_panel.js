/** @odoo-module **/

import {Component, useState} from "@odoo/owl";

export class EnvPanel extends Component {
    static template = "deploy.EnvPanel";
    static props = {
        envs: {type: Object},
        selected: {type: Object, optional: true},
        onSelect: {type: Function},
        undeployedBranches: {type: Array, optional: true},
        pendingProduction: {type: String, optional: true},
        pendingStaging: {type: Array, optional: true},
        onDeployBranch: {type: Function, optional: true},
    };

    setup() {
        this.state = useState({
            dragOverTarget: null,
        });

        this._onDragStart = (e, branch) => {
            e.dataTransfer.setData("text/plain", branch);
            e.dataTransfer.effectAllowed = "copy";
        };

        this._onDragOver = (e, target) => {
            e.preventDefault();
            e.dataTransfer.dropEffect = "copy";
            this.state.dragOverTarget = target;
        };

        this._onDragLeave = (e, target) => {
            if (this.state.dragOverTarget === target) {
                this.state.dragOverTarget = null;
            }
        };

        this._onDrop = (e, isProduction) => {
            e.preventDefault();
            this.state.dragOverTarget = null;
            const branch = e.dataTransfer.getData("text/plain");
            if (branch && this.props.onDeployBranch) {
                this.props.onDeployBranch(branch, isProduction);
            }
        };
    }
}
