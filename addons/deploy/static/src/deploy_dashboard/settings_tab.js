/** @odoo-module **/

import {Component, useState} from "@odoo/owl";
import {useService} from "@web/core/utils/hooks";

export class SettingsTab extends Component {
    static template = "deploy.SettingsTab";
    static props = {
        env: {type: Object, optional: true},
        agent: {type: Object, optional: true},
    };

    setup() {
        this.orm = useService("orm");
        this.notification = useService("notification");
        this.state = useState({
            editingWsUrl: false,
            wsUrlInput: "",
        });

        this.copyScript = () => {
            const el = document.createElement("textarea");
            el.value = this.props.agent?.bootstrap_script || "";
            document.body.appendChild(el);
            el.select();
            document.execCommand("copy");
            document.body.removeChild(el);
        };

        this.startEditWsUrl = () => {
            this.state.wsUrlInput = this.props.agent?.ws_url || "";
            this.state.editingWsUrl = true;
        };

        this.cancelEditWsUrl = () => {
            this.state.editingWsUrl = false;
        };

        this.saveWsUrl = async () => {
            const agentId = this.props.agent?.id;
            if (!agentId) return;
            try {
                await this.orm.write("deploy.agent", [agentId], {ws_url: this.state.wsUrlInput});
                this.props.agent.ws_url = this.state.wsUrlInput;
                this.state.editingWsUrl = false;
                this.notification.add("WebSocket URL saved", {type: "success"});
            } catch (e) {
                this.notification.add(e?.data?.message || e?.message || "Failed to save", {type: "danger"});
            }
        };
    }
}
