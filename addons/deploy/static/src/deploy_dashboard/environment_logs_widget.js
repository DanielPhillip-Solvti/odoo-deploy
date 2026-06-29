/** @odoo-module **/

import {Component, onPatched, onWillDestroy, onWillStart, useRef, useState} from "@odoo/owl";
import {registry} from "@web/core/registry";
import {rpc} from "@web/core/network/rpc";
import {standardFieldProps} from "@web/views/fields/standard_field_props";
import {useService} from "@web/core/utils/hooks";

export class EnvironmentLogs extends Component {
    static template = "deploy.EnvironmentLogs";
    static props = {...standardFieldProps};

    setup() {
        this.notification = useService("notification");
        this.orm = useService("orm");
        this.logContainer = useRef("logContainer");
        this.state = useState({
            lines: [],
            connected: false,
            streaming: false,
            autoScroll: true,
            agentId: null,
            wsUrl: "",
            branchName: "",
        });
        this._ws = null;

        this.toggleStream = () => {
            if (this.state.streaming) {
                this._disconnect();
            } else {
                this._connect();
            }
        };

        this.toggleAutoScroll = () => {
            this.state.autoScroll = !this.state.autoScroll;
        };

        this.clearLogs = () => {
            this.state.lines = [];
        };

        onWillStart(async () => {
            await this._loadMeta();
        });

        onWillDestroy(() => this._disconnect());

        onPatched(() => {
            if (this.state.autoScroll && this.logContainer.el) {
                this.logContainer.el.scrollTop = this.logContainer.el.scrollHeight;
            }
        });
    }

    async _loadMeta() {
        const envId = this.props.record.resId;
        if (!envId) return;
        try {
            const [env] = await this.orm.read("deploy.environment", [envId], ["agent_id", "repository_branch"]);
            const agentId = Array.isArray(env.agent_id) ? env.agent_id[0] : env.agent_id;
            this.state.branchName = env.repository_branch;
            if (!agentId) return;
            const [agent] = await this.orm.read("deploy.agent", [agentId], ["ws_url"]);
            this.state.agentId = agentId;
            this.state.wsUrl = agent.ws_url || "ws://localhost:9876";
        } catch (e) {
            // Silent
        }
    }

    async _connect() {
        const branch = this.state.branchName;
        const agentId = this.state.agentId;
        if (!branch || !agentId) return;

        this.state.streaming = true;
        this.state.connected = false;
        this.state.lines = [];
        try {
            const result = await rpc("/agent/ws/token", {
                agent_id: agentId,
                purpose: "logs",
                params: {branch},
            });

            if (result.error) {
                this.notification.add(result.error, {type: "danger"});
                this.state.streaming = false;
                return;
            }

            const wsUrl = result.ws_url || "ws://localhost:9876/logs-ws";
            this._ws = new WebSocket(`${wsUrl}?token=${result.token}`);

            this._ws.onopen = () => {
                this.state.connected = true;
            };

            this._ws.onmessage = (e) => {
                if (typeof e.data === "string") {
                    if (e.data === "DONE") {
                        this.state.connected = false;
                        this.state.streaming = false;
                        return;
                    }
                    if (e.data.startsWith("ERROR")) {
                        this.notification.add(e.data, {type: "danger"});
                        this._disconnect();
                        return;
                    }
                    this.state.lines = [...this.state.lines, e.data];
                }
            };

            this._ws.onerror = () => {
                this.notification.add("Log WebSocket connection failed", {type: "danger"});
                this._disconnect();
            };

            this._ws.onclose = () => {
                this.state.connected = false;
                this.state.streaming = false;
            };
        } catch (e) {
            this.notification.add(e?.data?.message || e?.message || "Failed to start log stream", {type: "danger"});
            this.state.streaming = false;
        }
    }

    _disconnect() {
        if (this._ws) {
            this._ws.close();
            this._ws = null;
        }
        this.state.connected = false;
        this.state.streaming = false;
    }
}

export const environmentLogs = {
    component: EnvironmentLogs,
    displayName: "Environment Logs",
    supportedTypes: ["char", "integer"],
};

registry.category("fields").add("deploy_environment_logs", environmentLogs);
