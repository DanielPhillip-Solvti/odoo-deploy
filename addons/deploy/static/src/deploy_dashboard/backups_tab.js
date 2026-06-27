/** @odoo-module **/

import {Component, useState} from "@odoo/owl";
import {useService} from "@web/core/utils/hooks";
import {rpc} from "@web/core/network/rpc";

export class BackupsTab extends Component {
    static template = "deploy.BackupsTab";
    static props = {
        agent: {type: Object, optional: true},
    };

    setup() {
        this.orm = useService("orm");
        this.notification = useService("notification");
        this.state = useState({
            selected: null,
            downloading: false,
            backingUp: false,
        });

        this.selectBackup = (filename) => {
            this.state.selected = filename;
        };

        this.downloadSelected = async () => {
            const filename = this.state.selected;
            if (!filename) return;

            const agentId = this.props.agent?.id;
            if (!agentId) return;

            this.state.downloading = true;
            try {
                const result = await rpc("/agent/backup/token", {
                    agent_id: agentId,
                    filename,
                });

                if (result.error) {
                    this.notification.add(result.error, {type: "danger"});
                    this.state.downloading = false;
                    return;
                }

                const wsUrl = result.ws_url || `ws://localhost:9876/backup-ws`;
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
            const agentId = this.props.agent?.id;
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
        if (!this.props.agent?.heartbeat_payload) return [];
        const payload =
            typeof this.props.agent.heartbeat_payload === "string"
                ? JSON.parse(this.props.agent.heartbeat_payload)
                : this.props.agent.heartbeat_payload;
        return payload.backups || [];
    }
}
