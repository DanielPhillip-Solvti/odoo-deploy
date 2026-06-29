/** @odoo-module **/

import {Component, onMounted, onWillStart, useState} from "@odoo/owl";
import {registry} from "@web/core/registry";
import {standardFieldProps} from "@web/views/fields/standard_field_props";
import {useService} from "@web/core/utils/hooks";

export class EnvironmentHistory extends Component {
    static template = "deploy.EnvironmentHistory";
    static props = {...standardFieldProps};

    setup() {
        this.orm = useService("orm");
        this.githubCache = useService("deploy_github_cache");
        this.state = useState({
            status: "loading",
            commits: [],
            errorMessage: "",
            branch: "",
            baseBranch: "",
            aheadBy: 0,
            behindBy: 0,
            mergeBaseCommit: null,
        });
        this._agentId = null;
        this._branchName = null;

        onWillStart(async () => {
            await this._loadMeta();
            this._loadFromCache();
        });

        onMounted(() => {
            if (this.state.status !== "loaded") {
                setTimeout(() => this._fetch(), 0);
            }
        });
    }

    async _loadMeta() {
        const envId = this.props.record.resId;
        if (!envId) return;
        try {
            const [env] = await this.orm.read("deploy.environment", [envId], ["agent_id", "repository_branch"]);
            this._agentId = Array.isArray(env.agent_id) ? env.agent_id[0] : env.agent_id;
            this._branchName = env.repository_branch;
        } catch (e) {
            this.state.status = "error";
            this.state.errorMessage = e.message;
        }
    }

    _loadFromCache() {
        if (!this._agentId || !this._branchName) return;
        const cached = this.githubCache.get(this._agentId, this._branchName);
        if (cached) {
            this.state.commits = cached.commits || [];
            this.state.branch = cached.branch || "";
            this.state.baseBranch = cached.base_branch || "";
            this.state.aheadBy = cached.ahead_by || 0;
            this.state.behindBy = cached.behind_by || 0;
            this.state.mergeBaseCommit = cached.merge_base_commit || null;
            this.state.status = "loaded";
        }
    }

    async _fetch() {
        const agentId = this._agentId;
        const branch = this._branchName;
        if (!agentId || !branch) {
            this.state.status = "no_record";
            return;
        }
        this.state.status = "loading";
        try {
            const result = await this.orm.call("deploy.agent", "get_github_commits", [agentId, branch]);
            if (result.error === "not_authenticated") {
                this.state.status = "not_authenticated";
            } else if (result.error) {
                this.state.status = "error";
                this.state.errorMessage = result.error;
            } else {
                this.state.commits = result.commits || [];
                this.state.branch = result.branch || "";
                this.state.baseBranch = result.base_branch || "";
                this.state.aheadBy = result.ahead_by || 0;
                this.state.behindBy = result.behind_by || 0;
                this.state.mergeBaseCommit = result.merge_base_commit || null;
                this.state.status = "loaded";
                this.githubCache.set(agentId, branch, result);
            }
        } catch (e) {
            this.state.status = "error";
            this.state.errorMessage = e.message || "Unknown error";
        }
    }
}

export const environmentHistory = {
    component: EnvironmentHistory,
    displayName: "Environment History",
    supportedTypes: ["char", "integer"],
};

registry.category("fields").add("deploy_environment_history", environmentHistory);
