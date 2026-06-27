/** @odoo-module **/

import {registry} from "@web/core/registry";
import {useService} from "@web/core/utils/hooks";
import {formView} from "@web/views/form/form_view";

export class AgentDashboardRedirectController extends formView.Controller {
    setup() {
        super.setup();
        const actionService = useService("action");
        const resId = this.props.resId || this.props.action?.params?.agent_id || this.props.action?.context?.active_id;
        if (resId) {
            actionService.doAction({
                type: "ir.actions.client",
                tag: "deploy.dashboard",
                params: {agent_id: resId},
                clearBreadcrumbs: true,
            });
        }
    }
}

registry.category("views").add("deploy.agent.dashboard", {
    ...formView,
    Controller: AgentDashboardRedirectController,
});
