/** @odoo-module **/

import {Component, onMounted, onWillDestroy, onWillStart, useRef, useState} from "@odoo/owl";
import {registry} from "@web/core/registry";
import {rpc} from "@web/core/network/rpc";
import {standardFieldProps} from "@web/views/fields/standard_field_props";
import {useService} from "@web/core/utils/hooks";

/* global Terminal, Uint8Array */

class FitAddon {
    activate(terminal) {
        this._terminal = terminal;
    }
    dispose() {
        // Noop
    }
    _measureCell() {
        // Try xterm.js internal renderer dimensions first
        const core = this._terminal._core;
        if (core) {
            const rs = core._renderService;
            if (rs && rs.dimensions && rs.dimensions.actualCellWidth && rs.dimensions.actualCellHeight) {
                return {w: rs.dimensions.actualCellWidth, h: rs.dimensions.actualCellHeight};
            }
        }
        // Fallback: measure a char element in the DOM
        const el = this._terminal.element;
        const measure = el && el.querySelector(".xterm-char-measure-element");
        if (measure) {
            const rect = measure.getBoundingClientRect();
            const text = measure.textContent || ">";
            const cw = rect.width / Math.max(text.length, 1);
            if (cw > 0 && rect.height > 0) return {w: cw, h: rect.height};
        }
        return null;
    }
    fit() {
        if (!this._terminal || !this._terminal.element) return;
        const parent = this._terminal.element.parentElement;
        if (!parent) return;
        const parentW = parent.clientWidth;
        const parentH = parent.clientHeight;
        if (parentW <= 0 || parentH <= 0) return;
        const cell = this._measureCell();
        if (!cell) return;
        const cols = Math.max(2, Math.floor(parentW / cell.w));
        const rows = Math.max(1, Math.floor(parentH / cell.h));
        if (this._terminal.rows === rows && this._terminal.cols === cols) return;
        this._terminal.resize(cols, rows);
    }
}

export class EnvironmentShell extends Component {
    static template = "deploy.EnvironmentShell";
    static props = {...standardFieldProps};

    setup() {
        this.notification = useService("notification");
        this.orm = useService("orm");
        this.termContainer = useRef("termContainer");
        this.state = useState({connected: false});
        this._term = null;
        this._fit = null;
        this._ws = null;
        this._token = null;
        this._agentId = null;
        this._wsUrl = "ws://localhost:9876";
        this._branchName = "";

        this.toggleConnection = () => {
            if (this._ws) {
                this._disconnect();
            } else {
                this._connect();
            }
        };

        onWillStart(async () => {
            await this._loadMeta();
        });

        onMounted(() => {
            this._term = new Terminal({
                cursorBlink: true,
                cursorStyle: "block",
                fontSize: 13,
                fontFamily: "'Fira Code', 'Cascadia Code', 'JetBrains Mono', 'Consolas', monospace",
                theme: {
                    background: "#1e1e1e",
                    foreground: "#d4d4d4",
                    cursor: "#d4d4d4",
                },
                allowProposedApi: true,
            });
            this._fit = new FitAddon();
            this._term.loadAddon(this._fit);
            this._term.open(this.termContainer.el);
            requestAnimationFrame(() => this._fit.fit());

            this._lastFit = 0;
            this._resizeObserver = new ResizeObserver(() => {
                const now = Date.now();
                if (this._fit && now - this._lastFit > 200) {
                    this._lastFit = now;
                    this._fit.fit();
                }
            });
            this._resizeObserver.observe(this.termContainer.el);
        });

        onWillDestroy(() => {
            this._disconnect();
            if (this._resizeObserver) this._resizeObserver.disconnect();
        });
    }

    get isConnected() {
        return this.state.connected;
    }

    async _loadMeta() {
        const envId = this.props.record.resId;
        if (!envId) return;
        try {
            const [env] = await this.orm.read("deploy.environment", [envId], ["agent_id", "repository_branch"]);
            const agentId = Array.isArray(env.agent_id) ? env.agent_id[0] : env.agent_id;
            this._branchName = env.repository_branch;
            if (!agentId) return;
            const [agent] = await this.orm.read("deploy.agent", [agentId], ["ws_url"]);
            this._agentId = agentId;
            this._wsUrl = agent.ws_url || "ws://localhost:9876";
        } catch (e) {
            // Silent
        }
    }

    async _connect() {
        const branch = this._branchName;
        const agentId = this._agentId;
        if (!branch || !agentId) return;

        try {
            const result = await rpc("/agent/ws/token", {
                agent_id: agentId,
                purpose: "shell",
                params: {branch},
            });

            if (result.error) {
                this.notification.add(result.error, {type: "danger"});
                return;
            }

            const wsUrl = result.ws_url ? result.ws_url.replace(/^http/, "ws") : "ws://localhost:9876/shell-ws";
            this._token = result.token;
            this._ws = new WebSocket(`${wsUrl}?token=${result.token}`);

            this._ws.onopen = () => {
                this.state.connected = true;
                this._term.clear();
                this._term.write("Connecting...\r\n");
                if (this._fit) this._fit.fit();
            };

            this._ws.onmessage = (e) => {
                if (e.data instanceof Blob) {
                    e.data.arrayBuffer().then((buf) => {
                        this._term.write(new Uint8Array(buf));
                    });
                } else if (typeof e.data === "string") {
                    if (e.data.startsWith("ERROR:")) {
                        this.notification.add(e.data, {type: "danger"});
                        return;
                    }
                    this._term.write(e.data);
                }
            };

            this._ws.onerror = () => {
                this.notification.add("Shell WebSocket connection failed", {type: "danger"});
                this._disconnect();
            };

            this._ws.onclose = () => {
                this.state.connected = false;
                this._term.write("\r\n\x1b[31m--- Connection closed ---\x1b[0m\r\n");
                this._ws = null;
            };

            this._term.onData((data) => {
                if (this._ws && this._ws.readyState === WebSocket.OPEN) {
                    this._ws.send(data);
                }
            });

            this._term.onResize(({cols, rows}) => {
                if (this._ws && this._ws.readyState === WebSocket.OPEN) {
                    this._ws.send(JSON.stringify({type: "resize", cols, rows}));
                }
            });

            window.addEventListener("resize", () => {
                if (this._fit) this._fit.fit();
            });
        } catch (e) {
            this.notification.add(e?.data?.message || e?.message || "Failed to start shell", {type: "danger"});
        }
    }

    _disconnect() {
        if (this._ws) {
            this._ws.close();
            this._ws = null;
        }
        this.state.connected = false;
        this._term.write("\r\n\x1b[33m--- Disconnected ---\x1b[0m\r\n");
    }
}

export const environmentShell = {
    component: EnvironmentShell,
    displayName: "Environment Shell",
    supportedTypes: ["char", "integer"],
};

registry.category("fields").add("deploy_environment_shell", environmentShell);
