/** @odoo-module **/

import {Component, onWillStart, useState} from "@odoo/owl";
import {registry} from "@web/core/registry";
import {useService} from "@web/core/utils/hooks";

export class GithubCommitHistory extends Component {
    static get template() {
        return "deploy.GithubCommitHistory";
    }
    static get props() {
        return {
            record: {type: Object},
            readonly: {type: Boolean, optional: true},
        };
    }

    setup() {
        this.orm = useService("orm");
        this.state = useState({
            status: "loading",
            commits: [],
            errorMessage: "",
        });

        onWillStart(() => this._loadCommits());
    }

    get environmentId() {
        return this.props.record.resId;
    }

    async _loadCommits() {
        if (!this.environmentId) {
            this.state.status = "no_record";
            return;
        }
        this.state.status = "loading";
        try {
            const result = await this.orm.call("deploy.environment", "get_github_commits", [[this.environmentId]]);
            if (result.error === "not_authenticated") {
                this.state.status = "not_authenticated";
            } else if (result.error) {
                this.state.status = "error";
                this.state.errorMessage = result.error;
            } else {
                this.state.commits = result.commits;
                this.state.status = "loaded";
            }
        } catch (e) {
            this.state.status = "error";
            this.state.errorMessage = e.message || "Unknown error";
        }
    }

    async reload() {
        await this._loadCommits();
    }

    formatDate(isoDate) {
        return new Date(isoDate).toLocaleString();
    }
}

registry.category("view_widgets").add("github_commit_history", {
    component: GithubCommitHistory,
});
