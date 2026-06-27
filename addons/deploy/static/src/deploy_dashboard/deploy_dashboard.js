/** @odoo-module **/

import {Component, onWillDestroy, onWillStart, useState} from "@odoo/owl";
import {registry} from "@web/core/registry";
import {useService} from "@web/core/utils/hooks";
import {EnvPanel} from "./env_panel";
import {EnvContent} from "./env_content";

export class DeployDashboard extends Component {
    static template = "deploy.DeployDashboard";
    static components = {EnvPanel, EnvContent};
    static props = ["*"];

    setup() {
        this.orm = useService("orm");
        this.busService = useService("bus_service");
        this.notification = useService("notification");
        this.state = useState({
            agent: null,
            environments: [],
            selectedEnv: null,
            activeTab: "history",
            heartbeat: null,
            loading: true,
            leftPanelOpen: false,
            undeployedBranches: [],
            pendingDeploys: [],
        });
        this._busHandler = null;
        this._eventCallbackHandler = null;
        this._busChannel = null;
        this._pendingEvents = {};
        this._pendingUndeploys = new Set();

        this.toggleLeftPanel = () => {
            this.state.leftPanelOpen = !this.state.leftPanelOpen;
        };

        this._parseEnvironments = (payload) => {
            if (!payload) return [];
            const envs = [];
            const pb = payload.production_branch;
            if (pb?.branch) {
                envs.push({
                    id: "prod-" + pb.branch,
                    repository_branch: pb.branch,
                    odoo_version: pb.odoo_version || "",
                    is_production: true,
                    state: pb.status || "",
                });
            }
            for (const sb of payload.staging_branches || []) {
                if (sb.branch) {
                    envs.push({
                        id: "staging-" + sb.branch,
                        repository_branch: sb.branch,
                        odoo_version: sb.odoo_version || "",
                        is_production: false,
                        state: sb.status || "",
                    });
                }
            }
            return envs;
        };

        this.selectEnv = (env) => {
            this.state.selectedEnv = env;
            if (window.innerWidth < 992) {
                this.state.leftPanelOpen = false;
            }
        };
        this.selectTab = (tabId) => {
            this.state.activeTab = tabId;
        };
        this.undeployBranch = async (branch) => {
            const agentId = this.state.agent?.id;
            if (!agentId) return;
            const env = this.state.environments.find((e) => e.repository_branch === branch);
            this._pendingUndeploys.add(branch);
            this.state.environments = this.state.environments.filter((e) => e.repository_branch !== branch);
            if (!this.state.undeployedBranches.includes(branch)) {
                this.state.undeployedBranches = [...this.state.undeployedBranches, branch];
            }
            if (this.state.selectedEnv?.repository_branch === branch) {
                this.state.selectedEnv = this.state.environments[0] || null;
            }
            try {
                const result = await this.orm.call("deploy.agent", "undeploy_branch", [agentId, branch]);
                if (result?.event_id) {
                    this._pendingEvents[result.event_id] = {
                        branch,
                        action: "undeploy",
                        isProduction: env?.is_production,
                        odooVersion: env?.odoo_version,
                    };
                }
            } catch (e) {
                this._pendingUndeploys.delete(branch);
                this.state.undeployedBranches = this.state.undeployedBranches.filter((b) => b !== branch);
                if (env) {
                    this.state.environments = [...this.state.environments, env];
                }
                const msg = e?.data?.message || e?.message || `Failed to undeploy ${branch}`;
                this.notification.add(msg, {type: "danger"});
            }
        };
        this.deployBranch = async (branch, isProduction) => {
            const agentId = this.state.agent?.id;
            if (!agentId) return;
            if (this.state.pendingDeploys.some((d) => d.branch === branch)) return;
            if (this.state.environments.some((e) => e.repository_branch === branch)) {
                this.notification.add(`Branch "${branch}" is already deployed.`, {type: "warning"});
                return;
            }
            if (isProduction && this.state.environments.some((e) => e.is_production)) {
                this.notification.add("A production environment already exists. Only one production branch is allowed.", {type: "warning"});
                return;
            }
            this.state.undeployedBranches = this.state.undeployedBranches.filter((b) => b !== branch);
            this.state.pendingDeploys = [...this.state.pendingDeploys, {branch, isProduction}];
            try {
                const result = await this.orm.call("deploy.agent", "deploy_branch", [agentId, branch, isProduction]);
                if (result?.event_id) {
                    this._pendingEvents[result.event_id] = {branch, isProduction, action: "deploy"};
                }
            } catch (e) {
                this.state.pendingDeploys = this.state.pendingDeploys.filter((d) => d.branch !== branch);
                this.state.environments = this.state.environments.filter((ee) => ee.repository_branch !== branch);
                if (!this.state.undeployedBranches.includes(branch)) {
                    this.state.undeployedBranches = [...this.state.undeployedBranches, branch];
                }
                const msg = e?.data?.message || e?.message || `Failed to deploy ${branch}`;
                this.notification.add(msg, {type: "danger"});
            }
        };

        onWillStart(() => this._init());
        onWillDestroy(() => this._cleanup());
    }

    async _init() {
        const agentId = this.props.action?.params?.agent_id || this.props.action?.context?.active_id || this._getAgentIdFromUrl();
        if (!agentId) {
            this.state.loading = false;
            return;
        }
        this._busChannel = `deploy_agent_${agentId}`;
        this.busService.addChannel(this._busChannel);
        this._busHandler = (payload) => this._onHeartbeat(payload);
        this.busService.subscribe("deploy.heartbeat", this._busHandler);
        this._eventCallbackHandler = (payload) => this._onEventCallback(payload);
        this.busService.subscribe("deploy.event_callback", this._eventCallbackHandler);
        this.busService.start();
        await this._loadAgent(agentId);
    }

    _cleanup() {
        if (this._busChannel) {
            this.busService.deleteChannel(this._busChannel);
        }
        if (this._busHandler) {
            this.busService.unsubscribe("deploy.heartbeat", this._busHandler);
        }
        if (this._eventCallbackHandler) {
            this.busService.unsubscribe("deploy.event_callback", this._eventCallbackHandler);
        }
    }

    async _loadAgent(agentId) {
        try {
            const agent = await this.orm.read(
                "deploy.agent",
                [agentId],
                ["name", "status", "repository_url", "last_heartbeat", "heartbeat_payload", "bootstrap_script", "ws_url"]
            );
            if (!agent.length) {
                this.state.loading = false;
                return;
            }
            this.state.agent = agent[0];
            const payload = agent[0].heartbeat_payload;
            if (payload) {
                const raw = typeof payload === "string" ? JSON.parse(payload) : payload;
                const envs = this._parseEnvironments(raw);
                this.state.environments = envs;
                const prod = envs.find((e) => e.is_production);
                this.state.selectedEnv = prod || envs[0];
            }
            this._loadUndeployedBranches(agentId);
        } catch (e) {
            // Silent
        }
        this.state.loading = false;
    }

    async _loadUndeployedBranches(agentId) {
        try {
            const branches = await this.orm.call("deploy.agent", "get_undeployed_branches", [agentId]);
            this.state.undeployedBranches = branches || [];
        } catch (e) {
            this.state.undeployedBranches = [];
        }
    }

    _onEventCallback(payload) {
        const pending = this._pendingEvents[payload.event_id];
        if (!pending) return;
        delete this._pendingEvents[payload.event_id];
        const branch = payload.branch || pending.branch;

        if (pending.action === "undeploy") {
            this._pendingUndeploys.delete(branch);
            if (payload.status === "success") {
                this.notification.add(`Branch ${branch} undeployed successfully.`, {
                    title: "Undeploy Complete",
                    type: "success",
                    sticky: false,
                });
            } else {
                this.state.undeployedBranches = this.state.undeployedBranches.filter((b) => b !== branch);
                if (pending.isProduction !== undefined) {
                    this.state.environments = [
                        ...this.state.environments,
                        {
                            id: (pending.isProduction ? "prod-" : "staging-") + branch,
                            repository_branch: branch,
                            odoo_version: pending.odooVersion || "",
                            is_production: pending.isProduction,
                            state: "active",
                        },
                    ];
                }
                this.notification.add(payload.message || `Undeploy of ${branch} failed.`, {
                    title: "Undeploy Failed",
                    type: "danger",
                    sticky: false,
                });
            }
            return;
        }

        this.state.pendingDeploys = this.state.pendingDeploys.filter((d) => d.branch !== branch);
        if (payload.status === "success") {
            if (!this.state.environments.some((e) => e.repository_branch === branch)) {
                const newEnv = {
                    id: (pending.isProduction ? "prod-" : "staging-") + branch,
                    repository_branch: branch,
                    odoo_version: "",
                    is_production: pending.isProduction,
                    state: "active",
                };
                this.state.environments = [...this.state.environments, newEnv];
                if (!this.state.selectedEnv) {
                    this.state.selectedEnv = newEnv;
                }
            }
            this.notification.add(`Branch ${branch} deployed successfully.`, {
                title: "Deploy Complete",
                type: "success",
                sticky: false,
            });
        } else {
            this.state.environments = this.state.environments.filter((e) => e.repository_branch !== branch);
            this.notification.add(payload.message || `Deployment of ${branch} failed.`, {
                title: "Deploy Failed",
                type: "danger",
                sticky: false,
            });
            if (!this.state.undeployedBranches.includes(branch)) {
                this.state.undeployedBranches = [...this.state.undeployedBranches, branch];
            }
        }
    }

    _getAgentIdFromUrl() {
        const hash = window.location.hash;
        const match = hash.match(/[?&]agent_id=(\d+)/);
        return match ? parseInt(match[1]) : undefined;
    }

    _onHeartbeat(payload) {
        this.state.heartbeat = payload;
        if (this.state.agent) {
            this.state.agent.status = payload.status;
            if (this.state.agent.heartbeat_payload) {
                const raw =
                    typeof this.state.agent.heartbeat_payload === "string"
                        ? JSON.parse(this.state.agent.heartbeat_payload)
                        : this.state.agent.heartbeat_payload;
                raw.backups = payload.backups || [];
                this.state.agent.heartbeat_payload = raw;
            }
        }
        if (payload.environments) {
            this.state.environments = payload.environments
                .map((e) => ({
                    id: (e.is_production ? "prod-" : "staging-") + e.branch,
                    repository_branch: e.branch,
                    odoo_version: e.odoo_version || "",
                    is_production: e.is_production,
                    state: e.status || "",
                }))
                .filter((e) => !this._pendingUndeploys.has(e.repository_branch));
            if (!this.state.selectedEnv) {
                const prod = this.state.environments.find((e) => e.is_production);
                this.state.selectedEnv = prod || this.state.environments[0];
            }
        }
    }

    get prodEnv() {
        return this.state.environments.find((e) => e.is_production);
    }

    get stagingEnvs() {
        return this.state.environments.filter((e) => !e.is_production);
    }

    get pendingProduction() {
        const pending = this.state.pendingDeploys.find((d) => d.isProduction);
        return pending ? pending.branch : undefined;
    }

    get pendingStaging() {
        return this.state.pendingDeploys.filter((d) => !d.isProduction).map((d) => d.branch);
    }

    get tabs() {
        return [
            {id: "history", label: "History", icon: "fa-history"},
            {id: "logs", label: "Logs", icon: "fa-file-text-o"},
            {id: "shell", label: "Shell", icon: "fa-terminal"},
            {id: "monitor", label: "Monitor", icon: "fa-heartbeat"},
            {id: "backups", label: "Backups", icon: "fa-database"},
            {id: "settings", label: "Settings", icon: "fa-gear"},
        ];
    }
}

registry.category("actions").add("deploy.dashboard", DeployDashboard);
