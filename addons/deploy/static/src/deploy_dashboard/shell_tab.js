/** @odoo-module **/

import {Component, onMounted, onWillDestroy, useRef} from "@odoo/owl";
import {useService} from "@web/core/utils/hooks";
import {rpc} from "@web/core/network/rpc";

/* global Terminal */

// Inline FitAddon — measures cell size from a canvas element in the terminal
class FitAddon {
    activate(terminal) {
        this._terminal = terminal;
    }
    dispose() {
        /* Required by addon interface */
    }
    fit() {
        if (!this._terminal || !this._terminal.element) return;
        const parent = this._terminal.element.parentElement;
        if (!parent) return;
        const parentW = parent.clientWidth;
        const parentH = parent.clientHeight;
        if (parentW <= 0 || parentH <= 0) return;

        const canvas = this._terminal.element.querySelector("canvas");
        if (!canvas) return;
        const cellW = canvas.width / this._terminal.cols;
        const cellH = canvas.height / this._terminal.rows;
        if (!cellW || !cellH || isNaN(cellW) || isNaN(cellH)) return;

        const cols = Math.max(2, Math.floor(parentW / cellW));
        const rows = Math.max(1, Math.floor(parentH / cellH));
        if (this._terminal.rows === rows && this._terminal.cols === cols) return;
        this._terminal.resize(cols, rows);
    }
}

export class ShellTab extends Component {
    static template = "deploy.ShellTab";
    static props = {
        env: {type: Object},
        agent: {type: Object, optional: true},
    };

    setup() {
        this.notification = useService("notification");
        this.termContainer = useRef("termContainer");
        this._term = null;
        this._fit = null;
        this._ws = null;
        this._token = null;

        this.toggleConnection = () => {
            if (this._ws) {
                this._disconnect();
            } else {
                this._connect();
            }
        };

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
            this._fit.fit();
        });

        onWillDestroy(() => this._disconnect());
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
                    this._ws.send(
                        JSON.stringify({
                            type: "resize",
                            cols: cols,
                            rows: rows,
                        })
                    );
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
        this._term.write("\r\n\x1b[33m--- Disconnected ---\x1b[0m\r\n");
    }
}
