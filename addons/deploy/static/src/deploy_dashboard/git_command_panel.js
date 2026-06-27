/** @odoo-module **/

import {Component, onWillStart, onWillUpdateProps, useState} from "@odoo/owl";
import {useService} from "@web/core/utils/hooks";

export class GitCommandPanel extends Component {
    static template = "deploy.GitCommandPanel";
    static props = {env: {type: Object}, agent: {type: Object}};

    setup() {
        this.orm = useService("orm");
        this.notification = useService("notification");
        this.state = useState({commands: [], selected: "clone"});

        this.select = (key) => {
            this.state.selected = key;
        };

        this.copy = () => {
            const cmd = this.state.commands.find((c) => c.key === this.state.selected);
            if (cmd) {
                this._copyText(cmd.command);
            }
        };

        this._copyText = (text) => {
            const textarea = document.createElement("textarea");
            textarea.value = text;
            textarea.style.position = "fixed";
            textarea.style.opacity = "0";
            document.body.appendChild(textarea);
            textarea.select();
            try {
                document.execCommand("copy");
                this.notification.add("Copied to clipboard", {type: "info"});
            } catch (e) {
                this.notification.add("Failed to copy", {type: "danger"});
            }
            document.body.removeChild(textarea);
        };

        onWillStart(() => this._loadCommands());
        onWillUpdateProps((nextProps) => {
            if (nextProps.env?.id !== this.props.env?.id) {
                this._loadCommands(nextProps.env);
            }
        });
    }

    async _loadCommands(env) {
        env = env || this.props.env;
        if (!env || !this.props.agent) return;
        const agentId = this.props.agent.resId || this.props.agent.id;
        const branch = env.repository_branch;
        const cmds = await this.orm.call("deploy.agent", "get_git_commands", [agentId, branch]);
        this.state.commands = cmds;
    }

    get currentCommand() {
        const cmd = this.state.commands.find((c) => c.key === this.state.selected);
        return cmd ? cmd.command : "";
    }
}
