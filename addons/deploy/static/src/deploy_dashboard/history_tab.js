/** @odoo-module **/

import {Component, onWillStart, onWillUpdateProps, useState} from "@odoo/owl";
import {useService} from "@web/core/utils/hooks";

export class HistoryTab extends Component {
    static template = "deploy.HistoryTab";
    static props = {
        env: {type: Object},
        agent: {type: Object},
    };

    setup() {
        this.orm = useService("orm");
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
        onWillStart(() => this._load());
        onWillUpdateProps((nextProps) => {
            if (nextProps.env?.id !== this.props.env?.id) {
                this._load(nextProps.env);
            }
        });
    }

    async _load(env) {
        env = env || this.props.env;
        const agent = this.props.agent;
        if (!agent || !env?.repository_branch) {
            this.state.status = "no_record";
            return;
        }
        const agentId = agent.resId || agent.id;
        const branch = env.repository_branch;
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
            }
        } catch (e) {
            this.state.status = "error";
            this.state.errorMessage = e.message || "Unknown error";
        }
    }

    async reload() {
        await this._load();
    }

    formatDate(isoDate) {
        return new Date(isoDate).toLocaleString();
    }
}
