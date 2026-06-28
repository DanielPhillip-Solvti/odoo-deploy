/** @odoo-module **/

import {Component, onPatched, onWillDestroy, useRef, useState} from "@odoo/owl";
import {useService} from "@web/core/utils/hooks";
import {rpc} from "@web/core/network/rpc";

export class LogsTab extends Component {
    static template = "deploy.LogsTab";
    static props = {
        env: {type: Object},
        agent: {type: Object, optional: true},
    };

    setup() {
        this.notification = useService("notification");
        this.logContainer = useRef("logContainer");
        this.state = useState({
            lines: [],
            connected: false,
            streaming: false,
            autoScroll: true,
        });
        this._ws = null;

        this.toggleStream = async () => {
            if (this.state.streaming) {
                this._disconnect();
                return;
            }
            await this._connect();
        };

        this.toggleAutoScroll = () => {
            this.state.autoScroll = !this.state.autoScroll;
        };

        this.clearLogs = () => {
            this.state.lines = [];
        };

        onWillDestroy(() => this._disconnect());

        onPatched(() => {
            if (this.state.autoScroll && this.logContainer.el) {
                this.logContainer.el.scrollTop = this.logContainer.el.scrollHeight;
            }
        });
    }

    get branch() {
        return this.props.env?.repository_branch;
    }

    get wsBaseUrl() {
        const agent = this.props.agent;
        if (agent?.ws_url) {
            const base = agent.ws_url.replace(/\/backup-ws.*$/, "");
            return base || "ws://localhost:9876";
        }
        return "ws://localhost:9876";
    }

    async _connect() {
        const branch = this.branch;
        const agentId = this.props.agent?.id;
        if (!branch || !agentId) return;

        this.state.streaming = true;
        this.state.lines = [];
        try {
            const result = await rpc("/agent/logs/token", {
                agent_id: agentId,
                branch,
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
