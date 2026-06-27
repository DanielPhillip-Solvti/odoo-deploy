/** @odoo-module **/

import {Component} from "@odoo/owl";
import {HistoryTab} from "./history_tab";
import {LogsTab} from "./logs_tab";
import {ShellTab} from "./shell_tab";
import {MonitorTab} from "./monitor_tab";
import {BackupsTab} from "./backups_tab";
import {SettingsTab} from "./settings_tab";
import {GitCommandPanel} from "./git_command_panel";

export class EnvContent extends Component {
    static template = "deploy.EnvContent";
    static components = {HistoryTab, LogsTab, ShellTab, MonitorTab, BackupsTab, SettingsTab, GitCommandPanel};
    static props = {
        env: {type: Object, optional: true},
        agent: {type: Object, optional: true},
        tabs: {type: Array},
        activeTab: {type: String},
        heartbeat: {type: Object, optional: true},
        onSelectTab: {type: Function},
        onUndeployBranch: {type: Function},
    };

    setup() {
        this.undeploy = () => {
            const env = this.props.env;
            if (!env) return;
            const branch = env.repository_branch;
            if (!window.confirm(`Are you sure you want to undeploy the branch "${branch}"? This action cannot be undone.`)) return;
            this.props.onUndeployBranch(branch);
        };
    }
}
