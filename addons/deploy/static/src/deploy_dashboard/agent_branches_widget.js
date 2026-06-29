/** @odoo-module **/

import {Component, onWillStart, useState} from "@odoo/owl";
import {registry} from "@web/core/registry";
import {standardFieldProps} from "@web/views/fields/standard_field_props";
import {useService} from "@web/core/utils/hooks";

export class AgentBranches extends Component {
    static template = "deploy.AgentBranches";
    static props = {...standardFieldProps};

    setup() {
        this.orm = useService("orm");
        this.notification = useService("notification");
        this.state = useState({
            branches: [],
            loading: true,
            deploying: {},
        });

        this.deployBranch = async (branch, isProduction) => {
            const agentId = this.props && this.props.record && this.props.record.resId;
            if (!agentId || this.state.deploying[branch]) return;

            if (isProduction && this.state.branches.length === 0) return;
            this.state.deploying[branch] = true;
            try {
                await this.orm.call("deploy.agent", "deploy_branch", [agentId, branch, isProduction]);
                this.state.branches = this.state.branches.filter((b) => b !== branch);
                this.notification.add(`Deploying ${branch}...`, {type: "info"});
            } catch (e) {
                const msg = e?.data?.message || e?.message || `Failed to deploy ${branch}`;
                this.notification.add(msg, {type: "danger"});
            }
            delete this.state.deploying[branch];
        };

        onWillStart(async () => {
            const agentId = this.props && this.props.record && this.props.record.resId;
            if (!agentId) {
                this.state.loading = false;
                return;
            }
            try {
                const branches = await this.orm.call("deploy.agent", "get_undeployed_branches", [agentId]);
                this.state.branches = branches || [];
            } catch (e) {
                this.state.branches = [];
            }
            this.state.loading = false;
        });
    }
}

export const agentBranches = {
    component: AgentBranches,
    displayName: "Available Branches",
    supportedTypes: ["char", "integer"],
};

registry.category("fields").add("deploy_agent_branches", agentBranches);
