/** @odoo-module **/

import {Component, useState} from "@odoo/owl";
import {registry} from "@web/core/registry";
import {rpc} from "@web/core/network/rpc";
import {standardFieldProps} from "@web/views/fields/standard_field_props";
import {useService} from "@web/core/utils/hooks";

export class AgentBackups extends Component {
    static template = "deploy.AgentBackups";
    static props = {...standardFieldProps};

    setup() {
        this.orm = useService("orm");
        this.notification = useService("notification");
        this.state = useState({
            selected: null,
            downloading: false,
            backingUp: false,
        });

        this.downloadBackup = async (filename) => {
            const agentId = this.props && this.props.record && this.props.record.resId;
            if (!agentId) return;

            this.state.downloading = true;
            try {
                const result = await rpc("/agent/ws/token", {
                    agent_id: agentId,
                    purpose: "backup",
                    params: {filename},
                });

                if (result.error) {
                    this.notification.add(result.error, {type: "danger"});
                    this.state.downloading = false;
                    return;
                }

                const wsUrl = result.ws_url || "ws://localhost:9876/backup-ws";
                const ws = new WebSocket(`${wsUrl}?token=${result.token}`);
                const chunks = [];

                ws.onmessage = (e) => {
                    if (e.data instanceof Blob) {
                        chunks.push(e.data);
                    } else if (e.data === "DONE") {
                        const blob = new Blob(chunks);
                        const a = document.createElement("a");
                        a.href = URL.createObjectURL(blob);
                        a.download = filename;
                        a.click();
                        URL.revokeObjectURL(a.href);
                        ws.close();
                        this.state.downloading = false;
                    } else if (typeof e.data === "string" && e.data.startsWith("ERROR")) {
                        this.notification.add(e.data, {type: "danger"});
                        ws.close();
                        this.state.downloading = false;
                    }
                };

                ws.onerror = () => {
                    this.notification.add("WebSocket connection failed", {type: "danger"});
                    this.state.downloading = false;
                };
            } catch (e) {
                this.notification.add(e?.data?.message || e?.message || "Failed to start download", {type: "danger"});
                this.state.downloading = false;
            }
        };

        this.triggerBackup = async (withDump) => {
            const agentId = this.props && this.props.record && this.props.record.resId;
            if (!agentId || this.state.backingUp) return;
            this.state.backingUp = true;
            try {
                const method = withDump ? "backup_with_dump" : "backup_no_dump";
                await this.orm.call("deploy.agent", method, [agentId]);
                this.notification.add(`Backup${withDump ? " with dump" : ""} queued successfully.`, {type: "success"});
            } catch (e) {
                this.notification.add(e?.data?.message || e?.message || "Backup failed", {type: "danger"});
            }
            this.state.backingUp = false;
        };
    }

    get backups() {
        const data = this.props && this.props.record && this.props.record.data;
        const payload = data && data.heartbeat_payload;
        if (!payload) return [];
        const raw = typeof payload === "string" ? JSON.parse(payload) : payload;
        return raw.backups || [];
    }
}

export const agentBackups = {
    component: AgentBackups,
    displayName: "Agent Backups",
    supportedTypes: ["char", "integer"],
};

registry.category("fields").add("deploy_agent_backups", agentBackups);
